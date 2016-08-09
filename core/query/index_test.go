package query

import (
	"testing"

	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
	"chain/database/pg/pgtest"
)

func TestAnnotatedTxs(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)

	indexer := NewIndexer(db, &cos.FC{})
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
