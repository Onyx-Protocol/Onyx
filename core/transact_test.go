package core

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/generator"
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
	g := generator.New(c, nil, db)
	pinStore := pin.NewStore(db)
	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	coretest.CreatePins(ctx, t, pinStore)
	accounts.IndexAccounts(query.NewIndexer(db, c, pinStore))
	go accounts.ProcessBlocks(ctx)

	acc := coretest.CreateAccount(ctx, t, accounts, "", nil)

	assetID := coretest.CreateAsset(ctx, t, assets, nil, "", nil)
	assetAmt := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}

	source := txbuilder.Action(assets.NewIssueAction(assetAmt, nil))
	dest := accounts.NewControlAction(assetAmt, acc, nil)

	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{source, dest}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)

	txdata, err := bc.NewTxDataFromBytes(tmpl.RawTransaction)
	if err != nil {
		t.Fatal(err)
	}

	err = txbuilder.FinalizeTx(ctx, c, g, bc.NewTx(*txdata))
	if err != nil {
		t.Fatal(err)
	}
	b := prottest.MakeBlock(t, c, g.PendingTxs())
	if len(b.Transactions) != 1 {
		t.Errorf("len(b.Transactions) = %d, want 1", len(b.Transactions))
	}
	<-pinStore.PinWaiter(account.PinName, c.Height())

	// Add a new source, spending the change output produced above.
	source = accounts.NewSpendAction(assetAmt, acc, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{source, dest}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)

	txdata, err = bc.NewTxDataFromBytes(tmpl.RawTransaction)
	if err != nil {
		t.Fatal(err)
	}

	err = txbuilder.FinalizeTx(ctx, c, g, bc.NewTx(*txdata))
	if err != nil {
		t.Fatal(err)
	}
	b = prottest.MakeBlock(t, c, g.PendingTxs())
	if len(b.Transactions) != 1 {
		t.Errorf("len(b.Transactions) = %d, want 1", len(b.Transactions))
	}
}

func TestRecordSubmittedTxs(t *testing.T) {
	ctx := context.Background()
	dbtx := pgtest.NewTx(t)

	testCases := []struct {
		hash   bc.Hash
		height uint64
		want   uint64
	}{
		{hash: bc.Hash{0x01}, height: 2, want: 2},
		{hash: bc.Hash{0x02}, height: 3, want: 3},
		{hash: bc.Hash{0x01}, height: 3, want: 2},
	}

	for i, tc := range testCases {
		got, err := recordSubmittedTx(ctx, dbtx, tc.hash, tc.height)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("%d: got %d want %d for hash %s", i, got, tc.want, tc.hash)
		}
	}
}
