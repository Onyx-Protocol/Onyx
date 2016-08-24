package query

import (
	"context"
	"testing"

	"chain/database/pg/pgtest"
	"chain/protocol"
	"chain/protocol/bc"
)

func TestIndexBlock(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)

	indexer := NewIndexer(db, &protocol.FC{})
	b := &bc.Block{
		Transactions: []*bc.Tx{},
	}
	indexer.indexBlockCallback(ctx, b)

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

	indexer := NewIndexer(db, &protocol.FC{})
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
