package core

import (
	"context"
	"testing"
	"time"

	"chain-stealth/core/account"
	"chain-stealth/core/asset"
	"chain-stealth/core/confidentiality"
	"chain-stealth/core/coretest"
	"chain-stealth/core/pin"
	"chain-stealth/core/query"
	"chain-stealth/core/txbuilder"
	"chain-stealth/database/pg/pgtest"
	"chain-stealth/protocol/bc"
	"chain-stealth/protocol/mempool"
	"chain-stealth/protocol/prottest"
	"chain-stealth/testutil"
)

func TestAccountTransferSpendChange(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	p := mempool.New()
	pinStore := pin.NewStore(db)
	conf := &confidentiality.Storage{DB: db}
	assets := asset.NewRegistry(db, c, pinStore, conf)
	accounts := account.NewManager(db, c, pinStore, conf)
	coretest.CreatePins(ctx, t, pinStore)
	accounts.IndexAccounts(query.NewIndexer(db, c, pinStore))
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

	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{source, dest}, nil, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, p, bc.NewTx(*tmpl.Transaction))
	if err != nil {
		t.Fatal(err)
	}
	b := prottest.MakeBlock(t, c, p.Dump(ctx))
	if len(b.Transactions) != 1 {
		t.Errorf("len(b.Transactions) = %d, want 1", len(b.Transactions))
	}
	<-pinStore.PinWaiter(account.PinName, c.Height())

	// Add a new source, spending the change output produced above.
	source = accounts.NewSpendAction(assetAmt, acc.ID, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{source, dest}, nil, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, p, bc.NewTx(*tmpl.Transaction))
	if err != nil {
		t.Fatal(err)
	}
	b = prottest.MakeBlock(t, c, p.Dump(ctx))
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
