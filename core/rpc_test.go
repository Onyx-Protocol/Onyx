package core

import (
	"bytes"
	"context"
	"testing"

	"chain/core/txdb"
	"chain/database/pg/pgtest"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestGetBlock(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	store := txdb.NewStore(db)
	chain := prottest.NewChainWithStorage(t, store)
	api := &API{Chain: chain, Store: store}

	block, err := api.getBlockRPC(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if block == nil {
		t.Error("expected 1 (initial) block, got none")
	}

	newBlock := prottest.MakeBlock(t, chain, nil)
	buf := new(bytes.Buffer)
	_, err = newBlock.WriteTo(buf)
	if err != nil {
		t.Fatal(err)
	}

	block, err = api.getBlockRPC(ctx, 2)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if block == nil {
		t.Error("expected 1 block, got none")
	}
	if !bytes.Equal(block, buf.Bytes()) {
		t.Errorf("got=%x, want=%s", block, buf.Bytes())
	}
}
