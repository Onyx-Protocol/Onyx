package query

import (
	"context"
	"testing"

	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
)

func TestAnnotatedTxs(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)

	c := prottest.NewChain(t)
	indexer := NewIndexer(db, c, nil)
	b := &bc.Block{
		Transactions: []*bc.Tx{
			prottest.NewIssuanceTx(t, c),
			prottest.NewIssuanceTx(t, c),
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
