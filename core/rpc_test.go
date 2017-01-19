package core

import (
	"bytes"
	"context"
	"testing"

	"chain/core/pb"
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
	r := &Handler{Chain: chain, Store: store}

	resp, err := r.GetBlock(ctx, &pb.GetBlockRequest{Height: 1})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Block == nil {
		t.Error("expected 1 (initial) block, got none")
	}

	newBlock := prottest.MakeBlock(t, chain, nil)
	buf := new(bytes.Buffer)
	_, err = newBlock.WriteTo(buf)
	if err != nil {
		t.Fatal(err)
	}

	resp, err = r.GetBlock(ctx, &pb.GetBlockRequest{Height: 2})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if resp.Block == nil {
		t.Error("expected 1 block, got none")
	}
	if !bytes.Equal(resp.Block, buf.Bytes()) {
		t.Errorf("got=%x, want=%x", resp.Block, buf.Bytes())
	}
}
