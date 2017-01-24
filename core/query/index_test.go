package query

import (
	"context"
	"testing"

	"chain/database/pg/pgtest"
	"chain/protocol"
	"chain/protocol/bc"
)

func TestAnnotatedTxs(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)

	indexer := NewIndexer(db, &protocol.Chain{}, nil)
	b := &bc.Block{
		Transactions: []*bc.Tx{
			{TxHashes: bc.TxHashes{ID: bc.Hash{0: 0x01}}},
			{TxHashes: bc.TxHashes{ID: bc.Hash{0: 0x02}}},
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
