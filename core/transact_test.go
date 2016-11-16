package core

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestAccountTransferSpendChange(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	assets := asset.NewRegistry(db, c)
	accounts := account.NewManager(db, c)
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	accounts.IndexAccounts(query.NewIndexer(db, c, pinStore), pinStore)
	go accounts.ProcessBlocks(ctx)

	acc, err := accounts.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	assetID := coretest.CreateAsset(ctx, t, assets, nil, "", nil)
	assetAmt := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}

	source := txbuilder.Action(assets.NewIssueAction(assetAmt, nil))
	dest := accounts.NewControlAction(assetAmt, acc.ID, nil)

	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{source, dest}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, bc.NewTx(*tmpl.Transaction))
	if err != nil {
		t.Fatal(err)
	}
	b := prottest.MakeBlock(t, c)
	if len(b.Transactions) != 1 {
		t.Errorf("len(b.Transactions) = %d, want 1", len(b.Transactions))
	}
	<-pinStore.PinWaiter(account.PinName, c.Height())

	// Add a new source, spending the change output produced above.
	source = accounts.NewSpendAction(assetAmt, acc.ID, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{source, dest}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, bc.NewTx(*tmpl.Transaction))
	if err != nil {
		t.Fatal(err)
	}
	b = prottest.MakeBlock(t, c)
	if len(b.Transactions) != 1 {
		t.Errorf("len(b.Transactions) = %d, want 1", len(b.Transactions))
	}
}
