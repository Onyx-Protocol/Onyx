package account_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestAccountSourceReserve(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = context.Background()
		c        = prottest.NewChain(t)
		pinStore = pin.NewStore(db)
		accounts = account.NewManager(db, c, pinStore)
		assets   = asset.NewRegistry(db, c, pinStore)
		indexer  = query.NewIndexer(db, c, pinStore)

		accID = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		out   = coretest.IssueAssets(ctx, t, c, assets, accounts, asset, 2, accID)
	)

	coretest.CreatePins(ctx, t, pinStore)
	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go accounts.ProcessBlocks(ctx)
	prottest.MakeBlock(t, c)
	<-pinStore.PinWaiter(account.PinName, c.Height())

	assetAmount1 := bc.AssetAmount{
		AssetID: asset,
		Amount:  1,
	}
	source := accounts.NewSpendAction(assetAmount1, accID, nil, nil)

	var builder txbuilder.TemplateBuilder
	err := source.Build(ctx, time.Now().Add(time.Minute), &builder)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	tpl, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}

	wantTxIns := []*bc.TxInput{bc.NewSpendInput(out.Hash, out.Index, nil, out.AssetID, out.Amount, out.ControlProgram, nil)}
	if !reflect.DeepEqual(tpl.Transaction.Inputs, wantTxIns) {
		t.Errorf("build txins\ngot:\n\t%+v\nwant:\n\t%+v", tpl.Transaction.Inputs, wantTxIns)
	}
	if len(tpl.Transaction.Outputs) != 1 {
		t.Errorf("expected 1 change output")
	}
	if tpl.Transaction.Outputs[0].Amount != 1 {
		t.Errorf("expected change amount to be 1")
	}
	if !programInAccount(ctx, t, db, tpl.Transaction.Outputs[0].ControlProgram, accID) {
		t.Errorf("expected change control program to belong to account")
	}
}

func TestAccountSourceUTXOReserve(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = context.Background()
		c        = prottest.NewChain(t)
		pinStore = pin.NewStore(db)
		accounts = account.NewManager(db, c, pinStore)
		assets   = asset.NewRegistry(db, c, pinStore)
		indexer  = query.NewIndexer(db, c, pinStore)

		accID = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		out   = coretest.IssueAssets(ctx, t, c, assets, accounts, asset, 2, accID)
	)

	coretest.CreatePins(ctx, t, pinStore)
	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go accounts.ProcessBlocks(ctx)
	prottest.MakeBlock(t, c)
	<-pinStore.PinWaiter(account.PinName, c.Height())

	source := accounts.NewSpendUTXOAction(out.Outpoint)

	var builder txbuilder.TemplateBuilder
	err := source.Build(ctx, time.Now().Add(time.Minute), &builder)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	tpl, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}

	wantTxIns := []*bc.TxInput{bc.NewSpendInput(out.Hash, out.Index, nil, out.AssetID, out.Amount, out.ControlProgram, nil)}

	if !reflect.DeepEqual(tpl.Transaction.Inputs, wantTxIns) {
		t.Errorf("build txins\ngot:\n\t%+v\nwant:\n\t%+v", tpl.Transaction.Inputs, wantTxIns)
	}
}

func TestAccountSourceReserveIdempotency(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = context.Background()
		c        = prottest.NewChain(t)
		pinStore = pin.NewStore(db)
		accounts = account.NewManager(db, c, pinStore)
		assets   = asset.NewRegistry(db, c, pinStore)
		indexer  = query.NewIndexer(db, c, pinStore)

		accID        = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset        = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		_            = coretest.IssueAssets(ctx, t, c, assets, accounts, asset, 2, accID)
		_            = coretest.IssueAssets(ctx, t, c, assets, accounts, asset, 2, accID)
		assetAmount1 = bc.AssetAmount{
			AssetID: asset,
			Amount:  1,
		}

		// An idempotency key that both reservations should use.
		clientToken1 = "a-unique-idempotency-key"
		clientToken2 = "another-unique-idempotency-key"
		wantSrc      = accounts.NewSpendAction(assetAmount1, accID, nil, &clientToken1)
		gotSrc       = accounts.NewSpendAction(assetAmount1, accID, nil, &clientToken1)
		separateSrc  = accounts.NewSpendAction(assetAmount1, accID, nil, &clientToken2)
	)

	coretest.CreatePins(ctx, t, pinStore)
	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go accounts.ProcessBlocks(ctx)
	prottest.MakeBlock(t, c)
	<-pinStore.PinWaiter(account.PinName, c.Height())

	reserveFunc := func(source txbuilder.Action) []*bc.TxInput {
		var builder txbuilder.TemplateBuilder

		err := source.Build(ctx, time.Now().Add(time.Minute), &builder)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}
		tpl, err := builder.Build()
		if err != nil {
			t.Fatal(err)
		}
		if len(tpl.Transaction.Inputs) != 1 {
			t.Fatalf("got %d result utxo, expected 1 result utxo", len(tpl.Transaction.Inputs))
		}
		return tpl.Transaction.Inputs
	}

	var (
		got      = reserveFunc(gotSrc)
		want     = reserveFunc(wantSrc)
		separate = reserveFunc(separateSrc)
	)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reserve result\ngot:\n\t%+v\nwant:\n\t%+v", got, want)
	}

	// The third reservation attempt should be distinct and not the same as the first two.
	if reflect.DeepEqual(separate, want) {
		t.Errorf("reserve result\ngot:\n\t%+v\ndo not want:\n\t%+v", separate, want)
	}
}

func programInAccount(ctx context.Context, t testing.TB, db pg.DB, program []byte, account string) bool {
	const q = `SELECT signer_id=$1 FROM account_control_programs WHERE control_program=$2`
	var in bool
	err := db.QueryRow(ctx, q, account, program).Scan(&in)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return in
}
