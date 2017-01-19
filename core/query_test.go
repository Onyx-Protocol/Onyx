package core

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"chain/core/asset"
	"chain/core/pb"
	"chain/core/pin"
	"chain/core/query"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
)

func TestQueryWithClockSkew(t *testing.T) {
	ctx := context.Background()
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	c := prottest.NewChain(t)

	pinStore := pin.NewStore(db)
	err := pinStore.CreatePin(ctx, asset.PinName, 100)
	if err != nil {
		t.Fatal(err)
	}

	indexer := query.NewIndexer(db, c, pinStore)
	h := &Handler{DB: db, Chain: c, Indexer: indexer}

	tx := bc.NewTx(bc.TxData{})
	block := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Height:      100,
			TimestampMS: bc.Millis(time.Now().Add(time.Hour)),
		},
		Transactions: []*bc.Tx{tx},
	}
	err = indexer.IndexTransactions(ctx, block)
	if err != nil {
		t.Fatal(err)
	}

	p, err := h.ListTxs(ctx, &pb.ListTxsQuery{})
	if err != nil {
		t.Fatal(err)
	}

	var items []*query.AnnotatedTx
	err = json.Unmarshal(p.Items, &items)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Errorf("got=%d txs, want %d", len(items), 1)
	}
}
