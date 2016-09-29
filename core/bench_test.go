package core

import (
	"context"
	"testing"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/mockhsm"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/core/txdb"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

// Generate 10 2-by-2 issuance transactions per block,
// as fast as possible.
func BenchmarkIOUThroughput(b *testing.B) {
	const (
		txPerBlock  = 10
		setupBlocks = 60 * 5
	)

	b.StopTimer()

	url, db := pgtest.NewDB(b, pgtest.SchemaPath)
	b.Log("db url", url)
	ctx := pg.NewContext(context.Background(), db)
	store, pool := txdb.New(db)
	c := prottest.NewChainWithStorage(b, store, pool)
	b1, err := c.GetBlock(ctx, 1)
	if err != nil {
		testutil.FatalErr(b, err)
	}

	// Setup the transaction query indexer to index every transaction.
	indexer := query.NewIndexer(db, c)
	indexer.RegisterAnnotator(account.AnnotateTxs)
	indexer.RegisterAnnotator(asset.AnnotateTxs)
	asset.Init(c, indexer)
	account.Init(c, indexer)
	hsm := mockhsm.New(db)

	xpubA, err := hsm.CreateKey(ctx, "keyA")
	if err != nil {
		testutil.FatalErr(b, err)
	}

	xpubB, err := hsm.CreateKey(ctx, "keyB")
	if err != nil {
		testutil.FatalErr(b, err)
	}

	accA := testCreateAccount(ctx, b, xpubA.XPub, "accA")
	accB := testCreateAccount(ctx, b, xpubB.XPub, "accB")
	assetA := testCreateAsset(ctx, b, xpubA.XPub, "assetA", b1.Hash())
	assetB := testCreateAsset(ctx, b, xpubB.XPub, "assetB", b1.Hash())

	blockIter := func() {
		for j := 0; j < txPerBlock; j++ {
			tpl, err := buildSingle(ctx, &buildRequest{Actions: []*action{
				{assettest.NewIssueAction(bc.AssetAmount{assetA, 100}, nil)},
				{assettest.NewIssueAction(bc.AssetAmount{assetB, 100}, nil)},
				{assettest.NewAccountControlAction(bc.AssetAmount{assetA, 100}, accA, nil)},
				{assettest.NewAccountControlAction(bc.AssetAmount{assetB, 100}, accB, nil)},
			}})
			if err != nil {
				testutil.FatalErr(b, err)
			}

			xpubs := []string{xpubA.XPub.String()}
			err = txbuilder.Sign(ctx, tpl, xpubs, (&api{hsm: hsm}).mockhsmSignTemplate)
			if err != nil {
				testutil.FatalErr(b, err)
			}

			xpubs = []string{xpubB.XPub.String()}
			err = txbuilder.Sign(ctx, tpl, xpubs, (&api{hsm: hsm}).mockhsmSignTemplate)
			if err != nil {
				testutil.FatalErr(b, err)
			}

			// Can't just call submitSingle here because it waits for a block,
			// and we create the block below.
			tx := bc.NewTx(*tpl.Transaction)
			err = txbuilder.FinalizeTx(ctx, c, tx)
			if err != nil {
				testutil.FatalErr(b, err)
			}

			// Also call this (like finalizeTxWait does)
			err = account.IndexUnconfirmedUTXOs(ctx, tx)
			if err != nil {
				testutil.FatalErr(b, err)
			}
		}

		prottest.MakeBlock(ctx, b, c)
	}

	for i := 0; i < setupBlocks; i++ {
		blockIter()
	}

	b.StartTimer()
	for i := 0; i < b.N; i += txPerBlock {
		blockIter()
	}
}

func testCreateAccount(ctx context.Context, tb testing.TB, key chainkd.XPub, alias string) string {
	acc, err := account.Create(ctx, []string{key.String()}, 1, alias, nil, nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	return acc.ID
}

func testCreateAsset(ctx context.Context, tb testing.TB, key chainkd.XPub, alias string, b1Hash bc.Hash) bc.AssetID {
	asset, err := asset.Define(ctx, []string{key.String()}, 1, nil, b1Hash, alias, nil, nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	return asset.AssetID
}
