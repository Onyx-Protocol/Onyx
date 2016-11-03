package query

import (
	"context"
	"testing"

	"chain/core/coretest"
	"chain/core/pin"
	"chain/database/pg/pgtest"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/prottest"
)

func TestIndexBlock(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)
	c := prottest.NewChain(t)

	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	indexer := NewIndexer(db, c, pinStore)
	b := &bc.Block{
		Transactions: []*bc.Tx{},
	}
	indexer.IndexTransactions(ctx, b)

	var blockCount int
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM query_blocks").Scan(&blockCount)
	if err != nil {
		t.Fatal(err)
	}
	if blockCount != 1 {
		t.Errorf("got=%d annotated txs in db, want %d", blockCount, 1)
	}
}

func TestAnnotatedTxs(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)

	indexer := NewIndexer(db, &protocol.Chain{}, nil)
	b := &bc.Block{
		Transactions: []*bc.Tx{
			{Hash: bc.Hash{0: 0x01}},
			{Hash: bc.Hash{0: 0x02}},
		},
	}
	txs, err := indexer.insertAnnotatedTxs(ctx, b)
	if err != nil {
		t.Error(err)
	}
	if len(txs) != len(b.Transactions) {
		t.Errorf("Got %d transactions, expected %d", len(txs), len(b.Transactions))
	}
}
