package account_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/generator"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestAccountSourceReserve(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = context.Background()
		c        = prottest.NewChain(t)
		g        = generator.New(c, nil, db)
		pinStore = pin.NewStore(db)
		accounts = account.NewManager(db, c, pinStore)
		assets   = asset.NewRegistry(db, c, pinStore)
		indexer  = query.NewIndexer(db, c, pinStore)

		accID              = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset              = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		txOut, outEntry, _ = coretest.IssueAssets(ctx, t, c, g, assets, accounts, asset, 2, accID)
	)

	coretest.CreatePins(ctx, t, pinStore)
	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go accounts.ProcessBlocks(ctx)
	prottest.MakeBlock(t, c, g.PendingTxs())
	<-pinStore.PinWaiter(account.PinName, c.Height())

	assetAmount1 := bc.AssetAmount{
		AssetId: &asset,
		Amount:  1,
	}
	source := accounts.NewSpendAction(assetAmount1, accID, nil, nil)

	builder := txbuilder.NewBuilder(time.Now().Add(5 * time.Minute))
	err := source.Build(ctx, builder)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	_, tx, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}

	wantTxIns := []*legacy.TxInput{legacy.NewSpendInput(nil, *outEntry.Source.Ref, *txOut.AssetId, txOut.Amount, outEntry.Source.Position, txOut.ControlProgram, *outEntry.Data, nil)}
	if !testutil.DeepEqual(tx.Inputs, wantTxIns) {
		t.Errorf("build txins\ngot:\n\t%+v\nwant:\n\t%+v", tx.Inputs, wantTxIns)
	}
	if len(tx.Outputs) != 1 {
		t.Errorf("expected 1 change output")
	}
	if tx.Outputs[0].Amount != 1 {
		t.Errorf("expected change amount to be 1")
	}
	if !programInAccount(ctx, t, db, tx.Outputs[0].ControlProgram, accID) {
		t.Errorf("expected change control program to belong to account")
	}
}

func TestAccountSourceUTXOReserve(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = context.Background()
		c        = prottest.NewChain(t)
		g        = generator.New(c, nil, db)
		pinStore = pin.NewStore(db)
		accounts = account.NewManager(db, c, pinStore)
		assets   = asset.NewRegistry(db, c, pinStore)
		indexer  = query.NewIndexer(db, c, pinStore)

		accID                     = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset                     = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		txOut, outEntry, outputID = coretest.IssueAssets(ctx, t, c, g, assets, accounts, asset, 2, accID)
	)

	coretest.CreatePins(ctx, t, pinStore)
	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go accounts.ProcessBlocks(ctx)
	prottest.MakeBlock(t, c, g.PendingTxs())
	<-pinStore.PinWaiter(account.PinName, c.Height())

	source := accounts.NewSpendUTXOAction(outputID)

	builder := txbuilder.NewBuilder(time.Now().Add(5 * time.Minute))
	err := source.Build(ctx, builder)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	_, tx, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}

	wantTxIns := []*legacy.TxInput{legacy.NewSpendInput(nil, *outEntry.Source.Ref, *txOut.AssetId, txOut.Amount, outEntry.Source.Position, txOut.ControlProgram, *outEntry.Data, nil)}

	if !testutil.DeepEqual(tx.Inputs, wantTxIns) {
		t.Errorf("build txins\ngot:\n\t%+v\nwant:\n\t%+v", tx.Inputs, wantTxIns)
	}
}

func TestAccountSourceReserveIdempotency(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = context.Background()
		c        = prottest.NewChain(t)
		g        = generator.New(c, nil, db)
		pinStore = pin.NewStore(db)
		accounts = account.NewManager(db, c, pinStore)
		assets   = asset.NewRegistry(db, c, pinStore)
		indexer  = query.NewIndexer(db, c, pinStore)

		accID = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
	)

	coretest.IssueAssets(ctx, t, c, g, assets, accounts, asset, 2, accID)
	coretest.IssueAssets(ctx, t, c, g, assets, accounts, asset, 2, accID)

	var (
		assetAmount1 = bc.AssetAmount{
			AssetId: &asset,
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
	prottest.MakeBlock(t, c, g.PendingTxs())
	<-pinStore.PinWaiter(account.PinName, c.Height())

	reserveFunc := func(source txbuilder.Action) []*legacy.TxInput {
		builder := txbuilder.NewBuilder(time.Now().Add(5 * time.Minute))

		err := source.Build(ctx, builder)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		_, tx, err := builder.Build()
		if err != nil {
			t.Fatal(err)
		}
		if len(tx.Inputs) != 1 {
			t.Fatalf("got %d result utxo, expected 1 result utxo", len(tx.Inputs))
		}
		return tx.Inputs
	}

	var (
		got      = reserveFunc(gotSrc)
		want     = reserveFunc(wantSrc)
		separate = reserveFunc(separateSrc)
	)
	if !testutil.DeepEqual(got, want) {
		t.Errorf("reserve result\ngot:\n\t%+v\nwant:\n\t%+v", got, want)
	}

	// The third reservation attempt should be distinct and not the same as the first two.
	if testutil.DeepEqual(separate, want) {
		t.Errorf("reserve result\ngot:\n\t%+v\ndo not want:\n\t%+v", separate, want)
	}
}

func programInAccount(ctx context.Context, t testing.TB, db pg.DB, program []byte, account string) bool {
	const q = `SELECT signer_id=$1 FROM account_control_programs WHERE control_program=$2`
	var in bool
	err := db.QueryRowContext(ctx, q, account, program).Scan(&in)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return in
}
