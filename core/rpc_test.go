package core

import (
	"bytes"
	"context"
	"testing"

	"chain/core/txdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestGetBlocks(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	store, pool := txdb.New(db)
	chain := prottest.NewChainWithStorage(t, store, pool)
	a := &api{c: chain}

	blocks, err := a.getBlocksRPC(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 {
		t.Errorf("expected 1 (initial) block, got %d", len(blocks))
	}

	newBlock := prottest.MakeBlock(ctx, t, chain)
	buf := new(bytes.Buffer)
	_, err = newBlock.WriteTo(buf)
	if err != nil {
		t.Fatal(err)
	}

	blocks, err = a.getBlocksRPC(ctx, 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if len(blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocks))
	}
	if !bytes.Equal(blocks[0], buf.Bytes()) {
		t.Errorf("got=%x, want=%s", blocks[0], buf.Bytes())
	}
}
