package txdb

import (
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
	"chain/metrics"
	"chain/net/trace/span"
)

// loadPoolOutputs returns the outputs in 'load' that can be found.
// Entries from table pool_inputs that are spending blockchain
// outputs (rather than pool outputs) will have a zero value bc.Output field.
// If some are not found, they will be absent from the map
// (not an error).
func loadPoolOutputs(ctx context.Context, load []bc.Outpoint) (map[bc.Outpoint]*state.Output, error) {
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
	rows, err := pg.FromContext(ctx).Query(ctx, loadQ, pg.Strings(txHashes), pg.Uint32s(indexes))
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer rows.Close()
	outs := make(map[bc.Outpoint]*state.Output)
	for rows.Next() {
		o := new(state.Output)
		err := rows.Scan(
			&o.Outpoint.Hash,
			&o.Outpoint.Index,
			&o.AssetID,
			&o.Amount,
			&o.Script,
			&o.Metadata,
		)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		outs[o.Outpoint] = o
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err())
	}

	const spentQ = `
		SELECT tx_hash, index FROM pool_inputs
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	rows, err = pg.FromContext(ctx).Query(ctx, spentQ, pg.Strings(txHashes), pg.Uint32s(indexes))
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer rows.Close()
	for rows.Next() {
		var p bc.Outpoint
		err := rows.Scan(&p.Hash, &p.Index)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		if o := outs[p]; o != nil {
			o.Spent = true
		} else {
			outs[p] = &state.Output{Outpoint: p, Spent: true}
		}
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err())
	}

	return outs, nil
}

const poolUnspentP2COutputQuery = `
	SELECT tx_hash, index, asset_id, amount, script, metadata
	FROM utxos_status
	WHERE contract_hash = $1 AND asset_id = $2 AND NOT confirmed
	      AND (tx_hash, index) NOT IN (TABLE pool_inputs)
`

// utxoSet holds a set of utxo record values
// to be inserted into the db.
type utxoSet struct {
	txHash        pg.Strings
	index         pg.Uint32s
	assetID       pg.Strings
	amount        pg.Int64s
	addr          pg.Strings
	accountID     pg.Strings
	managerNodeID pg.Strings
	aIndex        pg.Int64s
	script        pg.Byteas
	metadata      pg.Byteas
	contractHash  pg.Byteas
}

func addToUTXOSet(set *utxoSet, out *Output) {
	set.txHash = append(set.txHash, out.Outpoint.Hash.String())
	set.index = append(set.index, out.Outpoint.Index)
	set.assetID = append(set.assetID, out.AssetID.String())
	set.amount = append(set.amount, int64(out.Amount))
	set.accountID = append(set.accountID, out.AccountID)
	set.managerNodeID = append(set.managerNodeID, out.ManagerNodeID)
	set.aIndex = append(set.aIndex, toKeyIndex(out.AddrIndex[:]))
	set.script = append(set.script, out.Script)
	set.metadata = append(set.metadata, out.Metadata)

	isPayToContract, contractHash, _ := txscript.TestPayToContract(out.Script)
	if isPayToContract {
		set.contractHash = append(set.contractHash, contractHash[:])
	} else {
		set.contractHash = append(set.contractHash, nil)
	}
}

func insertPoolTx(ctx context.Context, tx *bc.Tx) error {
	const q = `INSERT INTO pool_txs (tx_hash, data) VALUES ($1, $2)`
	_, err := pg.FromContext(ctx).Exec(ctx, q, tx.Hash, tx)
	return errors.Wrap(err)
}

func insertPoolOutputs(ctx context.Context, insert []*Output) error {
	var outs utxoSet
	for _, o := range insert {
		addToUTXOSet(&outs, o)
	}

	db := pg.FromContext(ctx)

	const q1 = `
		INSERT INTO utxos (
			tx_hash, index, asset_id, amount,
			script, contract_hash, metadata
		)
		SELECT
			unnest($1::text[]),
			unnest($2::bigint[]),
			unnest($3::text[]),
			unnest($4::bigint[]),
			unnest($5::bytea[]),
			unnest($6::bytea[]),
			unnest($7::bytea[])
	`
	_, err := db.Exec(ctx, q1,
		outs.txHash,
		outs.index,
		outs.assetID,
		outs.amount,
		outs.script,
		outs.contractHash,
		outs.metadata,
	)
	return err
}

// InsertAccountOutputs records the account data for utxos
// TODO: move this function outside of the txdb package
func InsertAccountOutputs(ctx context.Context, outs []*Output) error {
	var set utxoSet
	for _, out := range outs {
		addToUTXOSet(&set, out)
	}

	const q = `
		WITH outputs AS (
			SELECT t.* FROM unnest($1::text[], $2::bigint[], $3::text[], $4::bigint[], $5::text[], $6::text[], $7::bigint[])
			AS t(tx_hash, index, asset_id, amount, mnode, acc, addr_index)
			LEFT JOIN account_utxos a ON (t.tx_hash, t.index) = (a.tx_hash, a.index)
			WHERE a.tx_hash IS NULL
		)
		INSERT INTO account_utxos (tx_hash, index, asset_id, amount, manager_node_id, account_id, addr_index)
		SELECT * FROM outputs o
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q,
		set.txHash,
		set.index,
		set.assetID,
		set.amount,
		set.managerNodeID,
		set.accountID,
		set.aIndex,
	)

	return errors.Wrap(err)
}

// insertPoolInputs inserts outpoints into pool_inputs.
func insertPoolInputs(ctx context.Context, outs []bc.Outpoint) error {
	defer metrics.RecordElapsed(time.Now())
	var (
		txHashes []string
		index    []uint32
	)
	for _, o := range outs {
		txHashes = append(txHashes, o.Hash.String())
		index = append(index, o.Index)
	}

	const q = `
		INSERT INTO pool_inputs (tx_hash, index)
		SELECT unnest($1::text[]), unnest($2::bigint[])
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, pg.Strings(txHashes), pg.Uint32s(index))
	return errors.Wrap(err)
}

// CountPoolTxs returns the total number of unconfirmed transactions.
func CountPoolTxs(ctx context.Context) (uint64, error) {
	const q = `SELECT count(tx_hash) FROM pool_txs`
	var res uint64
	err := pg.FromContext(ctx).QueryRow(ctx, q).Scan(&res)
	return res, errors.Wrap(err)
}

func toKeyIndex(i []uint32) int64 {
	return int64(i[0])<<31 | int64(i[1]&0x7fffffff)
}
