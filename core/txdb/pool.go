package txdb

import (
	"context"

	"github.com/lib/pq"

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

// GetTxs looks up transactions by their hashes in the pool.
func (p *Pool) GetTxs(ctx context.Context, hashes ...bc.Hash) (map[bc.Hash]*bc.Tx, error) {
	return getPoolTxs(ctx, p.db, hashes...)
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

// Dump returns the pooled transactions in topological order.
func (p *Pool) Dump(ctx context.Context) ([]*bc.Tx, error) {
	return dumpPoolTxs(ctx, p.db)
}

// Clean removes txs from the pending tx pool.
func (p *Pool) Clean(ctx context.Context, txs []*bc.Tx) error {
	var deleteTxHashes []string
	for _, tx := range txs {
		deleteTxHashes = append(deleteTxHashes, tx.Hash.String())
	}

	// Delete pool_txs
	const txq = `DELETE FROM pool_txs WHERE tx_hash IN (SELECT unnest($1::text[]))`
	_, err := p.db.Exec(ctx, txq, pq.StringArray(deleteTxHashes))
	return errors.Wrap(err, "delete from pool_txs")
}

// CountTxs returns the total number of unconfirmed transactions. It
// is not a part of the Pool interface.
func (p *Pool) CountTxs(ctx context.Context) (uint64, error) {
	const q = `SELECT count(tx_hash) FROM pool_txs`
	var res uint64
	err := p.db.QueryRow(ctx, q).Scan(&res)
	return res, errors.Wrap(err)
}
