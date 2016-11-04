package query

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/pin"
	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func setupQueryTest(t *testing.T) (context.Context, *Indexer, time.Time, time.Time, *account.Account, *account.Account, *asset.Asset, *asset.Asset) {
	time1 := time.Now()

	dbURL, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	indexer := NewIndexer(db, c, pinStore)
	accounts := account.NewManager(db, c)
	assets := asset.NewRegistry(db, c)
	assets.IndexAssets(indexer, pinStore)
	indexer.RegisterAnnotator(accounts.AnnotateTxs)
	indexer.RegisterAnnotator(assets.AnnotateTxs)
	go pinStore.QueueBlocks(ctx, c, asset.PinName)
	go pinStore.QueueBlocks(ctx, c, TxPinName)
	go assets.ProcessBlocks(ctx, "testhost", dbURL)
	go indexer.ProcessBlocks(ctx, "testhost", dbURL)

	acct1, err := accounts.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	acct2, err := accounts.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	asset1Tags := map[string]interface{}{"currency": "USD"}

	asset1, err := assets.Define(ctx, []string{testutil.TestXPub.String()}, 1, nil, "", asset1Tags, nil)
	if err != nil {
		t.Fatal(err)
	}
	asset2, err := assets.Define(ctx, []string{testutil.TestXPub.String()}, 1, nil, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	coretest.IssueAssets(ctx, t, c, assets, accounts, asset1.AssetID, 867, acct1.ID)
	coretest.IssueAssets(ctx, t, c, assets, accounts, asset2.AssetID, 100, acct1.ID)

	prottest.MakeBlock(t, c)
	<-pinStore.WaitForPin(TxPinName, c.Height())

	time2 := time.Now()

	return ctx, indexer, time1, time2, acct1, acct2, asset1, asset2
}
