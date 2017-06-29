package generator

import (
	"context"
	"testing"

	"chain/database/pg/pgtest"
	"chain/protocol/bc/bcvm"
)

func TestSavePendingBlock(t *testing.T) {
	ctx := context.Background()
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)

	// Save a pending block.
	err := savePendingBlock(ctx, db, fakeBlock(100))
	if err != nil {
		t.Fatal(err)
	}

	// Saving another block at the same height or lower should error.
	err = savePendingBlock(ctx, db, fakeBlock(20))
	if err != errDuplicateBlock {
		t.Errorf("got %s, want %s", err, errDuplicateBlock)
	}
	err = savePendingBlock(ctx, db, fakeBlock(100))
	if err != errDuplicateBlock {
		t.Errorf("got %s, want %s", err, errDuplicateBlock)
	}

	// Saving a higher block should succeed.
	err = savePendingBlock(ctx, db, fakeBlock(101))
	if err != nil {
		t.Fatal(err)
	}
}

func fakeBlock(height uint64) *bcvm.Block {
	return &bcvm.Block{
		BlockHeader: bcvm.BlockHeader{Height: height},
	}
}
