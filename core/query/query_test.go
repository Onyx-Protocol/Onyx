package query

import (
	"context"
	"testing"
	"time"

	"chain-stealth/core/account"
	"chain-stealth/core/asset"
	"chain-stealth/core/confidentiality"
	"chain-stealth/core/coretest"
	"chain-stealth/core/pin"
	"chain-stealth/database/pg/pgtest"
	"chain-stealth/protocol/mempool"
	"chain-stealth/protocol/prottest"
	"chain-stealth/testutil"
)

func setupQueryTest(t *testing.T) (context.Context, *Indexer, time.Time, time.Time, *account.Account, *account.Account, *asset.Asset, *asset.Asset) {
	time1 := time.Now()

	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	indexer := NewIndexer(db, c, pinStore)
	conf := &confidentiality.Storage{DB: db}
	assets := asset.NewRegistry(db, c, pinStore, conf)
	accounts := account.NewManager(db, c, pinStore, conf)
	assets.IndexAssets(indexer)
	indexer.RegisterAnnotator(conf.AnnotateTxs)
	indexer.RegisterAnnotator(accounts.AnnotateTxs)
	indexer.RegisterAnnotator(assets.AnnotateTxs)
	go assets.ProcessBlocks(ctx)
	go accounts.ProcessBlocks(ctx)
	go indexer.ProcessBlocks(ctx)

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

	p := mempool.New()
	coretest.IssueAssets(ctx, t, c, p, assets, accounts, asset1.AssetID, 867, acct1.ID)
	coretest.IssueAssets(ctx, t, c, p, assets, accounts, asset2.AssetID, 100, acct1.ID)

	prottest.MakeBlock(t, c, p.Dump(ctx))
	<-pinStore.PinWaiter(TxPinName, c.Height())

	time2 := time.Now()

	return ctx, indexer, time1, time2, acct1, acct2, asset1, asset2
}
