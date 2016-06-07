package txdb

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/net/trace/span"
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
func (p *Pool) Insert(ctx context.Context, tx *bc.Tx, assets map[bc.AssetID]*state.AssetState) error {
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

	var outputs []*Output
	for i, out := range tx.Outputs {
		outputs = append(outputs, &Output{
			Output: state.Output{
				Outpoint: bc.Outpoint{Hash: tx.Hash, Index: uint32(i)},
				TxOutput: *out,
			},
		})
	}
	err = insertPoolOutputs(ctx, dbtx, outputs)
	if err != nil {
		return errors.Wrap(err, "insert into utxos")
	}

	err = addIssuances(ctx, dbtx, assets, false)
	if err != nil {
		return errors.Wrap(err, "adding issuances")
	}

	err = dbtx.Commit(ctx)
	return errors.Wrap(err, "committing database transaction")
}

// Dump returns the pooled transactions in topological order.
func (p *Pool) Dump(ctx context.Context) ([]*bc.Tx, error) {
	return dumpPoolTxs(ctx, p.db)
}

// GetPrevouts looks up all of the transaction's prevouts in the
// pool and returns any of the outputs that exist in the pool.
func (p *Pool) GetPrevouts(ctx context.Context, txs []*bc.Tx) (map[bc.Outpoint]*state.Output, error) {
	dbtx, err := p.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	ctx = pg.NewContext(ctx, dbtx)
	defer dbtx.Rollback(ctx)

	var prevouts []bc.Outpoint
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			prevouts = append(prevouts, in.Previous)
		}
	}

	outs, err := loadPoolOutputs(ctx, dbtx, prevouts)
	if err != nil {
		return outs, err
	}
	return outs, dbtx.Commit(ctx)
}

// Clean removes confirmedTxs and conflictTxs from the pending tx pool.
func (p *Pool) Clean(
	ctx context.Context,
	confirmedTxs, conflictTxs []*bc.Tx,
	assets map[bc.AssetID]*state.AssetState,
) error {
	dbtx, err := p.db.Begin(ctx)
	if err != nil {
		return errors.Wrap(err)
	}
	defer dbtx.Rollback(ctx)

	var (
		deleteTxHashes   []string
		conflictTxHashes []string
	)
	for _, tx := range append(confirmedTxs, conflictTxs...) {
		deleteTxHashes = append(deleteTxHashes, tx.Hash.String())
	}
	// TODO(kr): ideally there is no distinction between confirmedTxs
	// and conflictTxs here. We currently need to know the difference,
	// because we mix pool outputs with blockchain outputs in postgres,
	// and this means we have to take extra care not to delete confirmed
	// outputs.
	for _, tx := range conflictTxs {
		conflictTxHashes = append(conflictTxHashes, tx.Hash.String())
	}

	// Delete pool_txs
	const txq = `DELETE FROM pool_txs WHERE tx_hash IN (SELECT unnest($1::text[]))`
	_, err = dbtx.Exec(ctx, txq, pg.Strings(deleteTxHashes))
	if err != nil {
		return errors.Wrap(err, "delete from pool_txs")
	}

	// Delete pool outputs
	const outq = `
		DELETE FROM utxos u WHERE tx_hash IN (SELECT unnest($1::text[]))
	`
	_, err = dbtx.Exec(ctx, outq, pg.Strings(conflictTxHashes))
	if err != nil {
		return errors.Wrap(err, "delete from utxos")
	}

	err = setIssuances(ctx, dbtx, assets)
	if err != nil {
		return errors.Wrap(err, "removing issuances")
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

// loadPoolOutputs returns the outputs in 'load' that can be found.
// If some are not found, they will be absent from the map
// (not an error).
func loadPoolOutputs(ctx context.Context, dbtx *sql.Tx, load []bc.Outpoint) (map[bc.Outpoint]*state.Output, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		txHashes []string
		indexes  []uint32
	)
	for _, p := range load {
		txHashes = append(txHashes, p.Hash.String())
		indexes = append(indexes, p.Index)
	}

	const loadQ = `
		SELECT tx_hash, index, asset_id, amount, script, metadata
		  FROM utxos_status
		  WHERE NOT confirmed
		    AND (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	outs := make(map[bc.Outpoint]*state.Output)
	err := pg.ForQueryRows(pg.NewContext(ctx, dbtx), loadQ, pg.Strings(txHashes), pg.Uint32s(indexes), func(hash bc.Hash, index uint32, assetID bc.AssetID, amount uint64, script, metadata []byte) {
		o := &state.Output{
			Outpoint: bc.Outpoint{Hash: hash, Index: index},
			TxOutput: bc.TxOutput{
				AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: amount},
				Script:      script,
				Metadata:    metadata,
			},
		}
		outs[o.Outpoint] = o
	})
	if err != nil {
		return nil, err
	}
	return outs, err
}

// utxoSet holds a set of utxo record values
// to be inserted into the db.
type utxoSet struct {
	txHash   pg.Strings
	index    pg.Uint32s
	assetID  pg.Strings
	amount   pg.Int64s
	script   pg.Byteas
	metadata pg.Byteas
}

func addToUTXOSet(set *utxoSet, out *Output) {
	set.txHash = append(set.txHash, out.Outpoint.Hash.String())
	set.index = append(set.index, out.Outpoint.Index)
	set.assetID = append(set.assetID, out.AssetID.String())
	set.amount = append(set.amount, int64(out.Amount))
	set.script = append(set.script, out.Script)
	set.metadata = append(set.metadata, out.Metadata)
}

func insertPoolTx(ctx context.Context, db pg.DB, tx *bc.Tx) error {
	const q = `INSERT INTO pool_txs (tx_hash, data) VALUES ($1, $2)`
	_, err := db.Exec(ctx, q, tx.Hash, tx)
	return errors.Wrap(err)
}

func insertPoolOutputs(ctx context.Context, db pg.DB, insert []*Output) error {
	var outs utxoSet
	for _, o := range insert {
		addToUTXOSet(&outs, o)
	}

	const q1 = `
		INSERT INTO utxos (
			tx_hash, index, asset_id, amount,
			script, metadata
		)
		SELECT
			unnest($1::text[]),
			unnest($2::bigint[]),
			unnest($3::text[]),
			unnest($4::bigint[]),
			unnest($5::bytea[]),
			unnest($6::bytea[])
	`
	_, err := db.Exec(ctx, q1,
		outs.txHash,
		outs.index,
		outs.assetID,
		outs.amount,
		outs.script,
		outs.metadata,
	)
	return err
}
