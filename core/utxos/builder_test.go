package utxos

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/generator"
	"chain/core/pin"
	"chain/core/txbuilder"
	"chain/database/pg/pgtest"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestSpendUTXO(t *testing.T) {
	var (
		_, db     = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx       = context.Background()
		c         = prottest.NewChain(t)
		g         = generator.New(c, nil, db)
		pinStore  = pin.NewStore(db)
		accounts  = account.NewManager(db, c, pinStore)
		assets    = asset.NewRegistry(db, c, pinStore)
		utxoStore = &Store{DB: db, Chain: c, PinStore: pinStore}

		accID                  = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset                  = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		txOut, outEntry, outID = coretest.IssueAssets(ctx, t, c, g, assets, accounts, asset, 2, accID)
	)

	coretest.CreatePins(ctx, t, pinStore)
	err := pinStore.CreatePin(ctx, PinName, 0)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = pinStore.CreatePin(ctx, DeletePinName, 0)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	// Make a block so that account UTXOs are available to spend.
	go utxoStore.ProcessBlocks(ctx)
	txs := g.PendingTxs()
	prottest.MakeBlock(t, c, txs)
	<-pinStore.PinWaiter(PinName, c.Height())

	source := &spendUTXOAction{
		store:    utxoStore,
		OutputID: &outID,
	}

	builder := txbuilder.NewBuilder(time.Now().Add(5 * time.Minute))
	err = source.Build(ctx, builder)
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
