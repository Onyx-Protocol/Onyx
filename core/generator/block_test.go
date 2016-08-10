package generator_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/core/asset/assettest"
	"chain/core/txdb"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/testutil"
)

func TestGetAndAddBlockSignatures(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	store, pool := txdb.New(pg.FromContext(ctx).(*sql.DB))
	fc, g, err := assettest.InitializeSigningGenerator(ctx, store, pool)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	block, prev, err := fc.GenerateBlock(ctx, time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = g.GetAndAddBlockSignatures(ctx, block, prev)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	engine, err := txscript.NewEngineForBlock(prev.OutputScript, block, txscript.StandardVerifyFlags)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = engine.Execute()
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

func TestGetBlocks(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	store, pool := txdb.New(pg.FromContext(ctx).(*sql.DB))
	fc, g, err := assettest.InitializeSigningGenerator(ctx, store, pool)
	if err != nil {
		t.Fatal(err)
	}

	blocks, err := g.GetBlocks(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) != 1 {
		t.Errorf("expected 1 (genesis) block, got %d", len(blocks))
	}

	c := make(chan []*bc.Block)

	var innerErr error

	go func() {
		defer close(c)

		// expect this will wait until block 2 is ready
		blocks, err := g.GetBlocks(ctx, 1)
		if err == nil {
			c <- blocks
		} else {
			innerErr = err
		}
	}()

	assetID := assettest.CreateAssetFixture(ctx, t, nil, 0, nil, nil)
	assettest.IssueAssetsFixture(ctx, t, fc, assetID, 1, "")

	// Hopefully force the GetBlocks call to wait
	time.Sleep(10 * time.Millisecond)

	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	blocks, ok := <-c
	if !ok {
		t.Fatal(innerErr)
	}

	if len(blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Height != 2 {
		t.Errorf("expected block 2, got block %d", blocks[0].Height)
	}
}
