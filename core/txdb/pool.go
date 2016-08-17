package txdb

import (
	"context"

	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
)

// A Pool encapsulates storage of the pending transaction pool.
type Pool struct {
	db *sql.DB
}

// NewPool creates and returns a new Pool object.
//
// A Pool manages its own database transactions, so
// it requires a handle to a SQL database.
// For testing purposes, it is usually much faster
// and more convenient to use package chain/cos/mempool
// instead.
func NewPool(db *sql.DB) *Pool {
	return &Pool{db: db}
}

// GetTxs looks up transactions by their hashes in the pool.
func (p *Pool) GetTxs(ctx context.Context, hashes ...bc.Hash) (map[bc.Hash]*bc.Tx, error) {
	return getPoolTxs(ctx, p.db, hashes...)
}

// Insert adds the transaction to the pending pool.
func (p *Pool) Insert(ctx context.Context, tx *bc.Tx) error {
	dbtx, err := p.db.Begin(ctx)
	if err != nil {
		return errors.Wrap(err)
	}
	defer dbtx.Rollback(ctx)

	inserted, err := insertTx(ctx, dbtx, tx)
	if err != nil {
		return errors.Wrap(err, "insert into txs")
	}
	if !inserted {
		// Another SQL transaction already succeeded in applying the tx,
		// so there's no need to do anything else.
		return nil
	}

	err = insertPoolTx(ctx, dbtx, tx)
	if err != nil {
		return errors.Wrap(err, "insert into pool txs")
	}

	err = dbtx.Commit(ctx)
	return errors.Wrap(err, "committing database transaction")
}

// Dump returns the pooled transactions in topological order.
func (p *Pool) Dump(ctx context.Context) ([]*bc.Tx, error) {
	return dumpPoolTxs(ctx, p.db)
}

// Clean removes txs from the pending tx pool.
func (p *Pool) Clean(ctx context.Context, txs []*bc.Tx) error {
	dbtx, err := p.db.Begin(ctx)
	if err != nil {
		return errors.Wrap(err)
	}
	defer dbtx.Rollback(ctx)

	var deleteTxHashes []string
	for _, tx := range txs {
		deleteTxHashes = append(deleteTxHashes, tx.Hash.String())
	}

	// Delete pool_txs
	const txq = `DELETE FROM pool_txs WHERE tx_hash IN (SELECT unnest($1::text[]))`
	_, err = dbtx.Exec(ctx, txq, pg.Strings(deleteTxHashes))
	if err != nil {
		return errors.Wrap(err, "delete from pool_txs")
	}

	err = dbtx.Commit(ctx)
	return errors.Wrap(err, "pool update dbtx commit")
}

// CountTxs returns the total number of unconfirmed transactions. It
// is not a part of the cos.Pool interface.
func (p *Pool) CountTxs(ctx context.Context) (uint64, error) {
	const q = `SELECT count(tx_hash) FROM pool_txs`
	var res uint64
	err := p.db.QueryRow(ctx, q).Scan(&res)
	return res, errors.Wrap(err)
}

func insertPoolTx(ctx context.Context, db pg.DB, tx *bc.Tx) error {
	const q = `INSERT INTO pool_txs (tx_hash, data) VALUES ($1, $2)`
	_, err := db.Exec(ctx, q, tx.Hash, tx)
	return errors.Wrap(err)
}
