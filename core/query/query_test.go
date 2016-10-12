package query

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func setupQueryTest(t *testing.T) (context.Context, *Indexer, time.Time, time.Time, *account.Account, *account.Account, *asset.Asset, *asset.Asset) {
	time1 := time.Now()

	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	c := prottest.NewChain(t)
	indexer := NewIndexer(db, c)
	b1, err := c.GetBlock(ctx, 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	assets := asset.NewRegistry(c, b1.Hash())
	account.Init(c, indexer)
	indexer.RegisterAnnotator(account.AnnotateTxs)
	indexer.RegisterAnnotator(assets.AnnotateTxs)

	acct1, err := account.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	acct2, err := account.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
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

	assettest.IssueAssetsFixture(ctx, t, c, assets, asset1.AssetID, 867, acct1.ID)
	assettest.IssueAssetsFixture(ctx, t, c, assets, asset2.AssetID, 100, acct1.ID)

	prottest.MakeBlock(ctx, t, c)

	time2 := time.Now()

	return ctx, indexer, time1, time2, acct1, acct2, asset1, asset2
}
