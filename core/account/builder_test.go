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
		ctx      = pg.NewContext(context.Background(), db)
		c        = prottest.NewChain(t)
		accounts = account.NewManager(db, c)
		assets   = asset.NewRegistry(db, c, bc.Hash{})
		indexer  = query.NewIndexer(db, c)

		accID = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		out   = coretest.IssueAssets(ctx, t, c, assets, accounts, asset, 2, accID)
	)

	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	prottest.MakeBlock(ctx, t, c)

	assetAmount1 := bc.AssetAmount{
		AssetID: asset,
		Amount:  1,
	}
	source := accounts.NewSpendAction(assetAmount1, accID, nil, nil, nil, nil)

	buildResult, err := source.Build(ctx, time.Now().Add(time.Minute))
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	wantTxIns := []*bc.TxInput{bc.NewSpendInput(out.Hash, out.Index, nil, out.AssetID, out.Amount, out.ControlProgram, nil)}

	if !reflect.DeepEqual(buildResult.Inputs, wantTxIns) {
		t.Errorf("build txins\ngot:\n\t%+v\nwant:\n\t%+v", buildResult.Inputs, wantTxIns)
	}

	if len(buildResult.Outputs) != 1 {
		t.Errorf("expected 1 change output")
	}

	if buildResult.Outputs[0].Amount != 1 {
		t.Errorf("expected change amount to be 1")
	}

	if !programInAccount(ctx, t, db, buildResult.Outputs[0].ControlProgram, accID) {
		t.Errorf("expected change control program to belong to account")
	}
}

func TestAccountSourceUTXOReserve(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = pg.NewContext(context.Background(), db)
		c        = prottest.NewChain(t)
		assets   = asset.NewRegistry(db, c, bc.Hash{})
		accounts = account.NewManager(db, c)
		indexer  = query.NewIndexer(db, c)

		accID = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		out   = coretest.IssueAssets(ctx, t, c, assets, accounts, asset, 2, accID)
	)

	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	prottest.MakeBlock(ctx, t, c)

	source := accounts.NewSpendUTXOAction(out.Outpoint)
	buildResult, err := source.Build(ctx, time.Now().Add(time.Minute))
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	wantTxIns := []*bc.TxInput{bc.NewSpendInput(out.Hash, out.Index, nil, out.AssetID, out.Amount, out.ControlProgram, nil)}

	if !reflect.DeepEqual(buildResult.Inputs, wantTxIns) {
		t.Errorf("build txins\ngot:\n\t%+v\nwant:\n\t%+v", buildResult.Inputs, wantTxIns)
	}
}

func TestAccountSourceReserveIdempotency(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = pg.NewContext(context.Background(), db)
		c        = prottest.NewChain(t)
		assets   = asset.NewRegistry(db, c, bc.Hash{})
		accounts = account.NewManager(db, c)
		indexer  = query.NewIndexer(db, c)

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
		wantSrc      = accounts.NewSpendAction(assetAmount1, accID, nil, nil, nil, &clientToken1)
		gotSrc       = accounts.NewSpendAction(assetAmount1, accID, nil, nil, nil, &clientToken1)
		separateSrc  = accounts.NewSpendAction(assetAmount1, accID, nil, nil, nil, &clientToken2)
	)

	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	prottest.MakeBlock(ctx, t, c)

	reserveFunc := func(source txbuilder.Action) []*bc.TxInput {
		buildResult, err := source.Build(ctx, time.Now().Add(time.Minute))
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}
		if len(buildResult.Inputs) != 1 {
			t.Fatalf("expected 1 result utxo")
		}
		return buildResult.Inputs
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

func TestAccountSourceWithTxHash(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = pg.NewContext(context.Background(), db)
		c        = prottest.NewChain(t)
		assets   = asset.NewRegistry(db, c, bc.Hash{})
		accounts = account.NewManager(db, c)
		indexer  = query.NewIndexer(db, c)

		acc      = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset    = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		assetAmt = bc.AssetAmount{AssetID: asset, Amount: 1}
		utxos    = 4
		srcTxs   []bc.Hash
	)

	for i := 0; i < utxos; i++ {
		o := coretest.IssueAssets(ctx, t, c, assets, accounts, asset, 1, acc)
		srcTxs = append(srcTxs, o.Outpoint.Hash)
	}

	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	prottest.MakeBlock(ctx, t, c)

	for i := 0; i < utxos; i++ {
		theTxHash := srcTxs[i]
		source := accounts.NewSpendAction(assetAmt, acc, &theTxHash, nil, nil, nil)

		buildResult, err := source.Build(ctx, time.Now().Add(time.Minute))
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		if len(buildResult.Inputs) != 1 {
			t.Fatalf("expected 1 result utxo")
		}

		got := buildResult.Inputs[0].Outpoint()
		want := bc.Outpoint{Hash: theTxHash, Index: 0}
		if got != want {
			t.Errorf("reserved utxo outpoint got=%v want=%v", got, want)
		}
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
