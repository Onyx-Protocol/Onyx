package query

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/blocksigner"
	"chain/core/generator"
	"chain/core/mockhsm"
	"chain/core/txdb"
	"chain/cos"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func setupQueryTest(t *testing.T) (context.Context, *Indexer, time.Time, time.Time, *account.Account, *account.Account, *asset.Asset, *asset.Asset) {
	time1 := time.Now()

	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	store, pool := txdb.New(db)
	fc, err := cos.NewFC(ctx, store, pool, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	indexer := NewIndexer(db, fc)
	asset.Init(fc, indexer, true)
	account.Init(fc, indexer)
	indexer.RegisterAnnotator(account.AnnotateTxs)
	indexer.RegisterAnnotator(asset.AnnotateTxs)
	hsm := mockhsm.New(db)
	xpub, err := hsm.CreateKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	localSigner := blocksigner.New(xpub, hsm, db, fc)
	g := &generator.Generator{
		Config: generator.Config{
			LocalSigner:  localSigner,
			BlockPeriod:  time.Second,
			BlockKeys:    []ed25519.PublicKey{xpub.Key},
			SigsRequired: 1,
			FC:           fc,
		},
	}
	genesis, err := fc.UpsertGenesisBlock(ctx, []ed25519.PublicKey{xpub.Key}, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	genesisHash := genesis.Hash()

	acct1, err := account.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	acct2, err := account.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	asset1Tags := map[string]interface{}{"currency": "USD"}

	asset1, err := asset.Define(ctx, []string{testutil.TestXPub.String()}, 1, nil, genesisHash, asset1Tags, nil)
	if err != nil {
		t.Fatal(err)
	}
	asset2, err := asset.Define(ctx, []string{testutil.TestXPub.String()}, 1, nil, genesisHash, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	assettest.IssueAssetsFixture(ctx, t, fc, asset1.AssetID, 867, acct1.ID)

	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	time2 := time.Now()

	return ctx, indexer, time1, time2, acct1, acct2, asset1, asset2
}
