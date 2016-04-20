package generator_test

import (
	"testing"
	"time"

	"chain/api/asset/assettest"
	. "chain/api/generator"
	"chain/cos/bc"
	"chain/cos/txscript"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestGetAndAddBlockSignatures(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	fc, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	block, prev, err := fc.GenerateBlock(ctx, time.Now())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = GetAndAddBlockSignatures(ctx, block, prev)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	engine, err := txscript.NewEngineForBlock(ctx, prev.OutputScript, block, txscript.StandardVerifyFlags)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = engine.Execute()
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

func TestGetBlocks(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	_, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	blocks, err := GetBlocks(ctx, 0)
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
		blocks, err := GetBlocks(ctx, 1)
		if err == nil {
			c <- blocks
		} else {
			innerErr = err
		}
	}()

	assetID := assettest.CreateAssetFixture(ctx, t, "", "", "")
	assettest.IssueAssetsFixture(ctx, t, assetID, 1, "")

	// Hopefully force the GetBlocks call to wait
	time.Sleep(10 * time.Millisecond)

	_, err = MakeBlock(ctx)
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
