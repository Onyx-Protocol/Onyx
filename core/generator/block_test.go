package generator

import (
	"context"
	"testing"
	"time"

	"chain/core/asset/assettest"
	"chain/core/txdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/protocol/bc"
	"chain/protocol/prottest"
)

// TODO(kr): GetBlocks is not a generator function.
// Move this test (and GetBlocks) to another package.
func TestGetBlocks(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	store, pool := txdb.New(pg.FromContext(ctx).(*sql.DB))
	chain, err := assettest.InitializeSigningGenerator(ctx, store, pool)
	if err != nil {
		t.Fatal(err)
	}

	blocks, err := GetBlocks(ctx, chain, 0)
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
		blocks, err := GetBlocks(ctx, chain, 1)
		if err == nil {
			c <- blocks
		} else {
			innerErr = err
		}
	}()

	assetID := assettest.CreateAssetFixture(ctx, t, nil, 0, nil, "", nil)
	assettest.IssueAssetsFixture(ctx, t, chain, assetID, 1, "")

	// Hopefully force the GetBlocks call to wait
	time.Sleep(10 * time.Millisecond)

	prottest.MakeBlock(ctx, t, chain)

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
