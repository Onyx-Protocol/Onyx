package generator

import (
	"context"
	"testing"

	"chain/core/txdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/protocol/mempool"
	"chain/protocol/prottest"
	"chain/testutil"
)

// TODO(kr): GetBlocks is not a generator function.
// Move this test (and GetBlocks) to another package.
func TestGetBlocks(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	store := txdb.NewStore(db)
	chain := prottest.NewChainWithStorage(t, store, mempool.New())

	blocks, err := GetBlocks(ctx, chain, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) != 1 {
		t.Errorf("expected 1 (initial) block, got %d", len(blocks))
	}

	prottest.MakeBlock(ctx, t, chain)

	blocks, err = GetBlocks(ctx, chain, 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if len(blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Height != 2 {
		t.Errorf("expected block 2, got block %d", blocks[0].Height)
	}
}
