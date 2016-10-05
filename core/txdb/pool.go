package txdb

import (
	"context"

	"chain/database/pg"
	"chain/errors"
	"chain/protocol/bc"
)

// A Pool encapsulates storage of the pending transaction pool.
type Pool struct {
	db pg.DB
}

// NewPool creates and returns a new Pool object.
//
// For testing purposes, it is usually much faster
// and more convenient to use package chain/protocol/mempool
// instead.
func NewPool(db pg.DB) *Pool {
	return &Pool{db: db}
}

// Insert adds the transaction to the pending pool.
func (p *Pool) Insert(ctx context.Context, tx *bc.Tx) error {
	const q = `
		INSERT INTO pool_txs (tx_hash, data) VALUES ($1, $2)
		ON CONFLICT (tx_hash) DO NOTHING
	`
	_, err := p.db.Exec(ctx, q, tx.Hash, tx)
	return errors.Wrap(err, "insert into pool txs")
}

// Dump returns the pooled transactions in topological order and
// empties the pool.
func (p *Pool) Dump(ctx context.Context) ([]*bc.Tx, error) {
	const q = `DELETE FROM pool_txs RETURNING tx_hash, data`
	var txs []*bc.Tx
	err := pg.ForQueryRows(pg.NewContext(ctx, p.db), q, func(hash bc.Hash, data bc.TxData) {
		txs = append(txs, &bc.Tx{TxData: data, Hash: hash})
	})
	if err != nil {
		return nil, err
	}
	txs = topSort(ctx, txs)
	return txs, nil
}
