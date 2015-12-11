package txdb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/api/utxodb"
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
		    FROM utxos
		    WHERE NOT confirmed
		        AND (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	rows, err := pg.FromContext(ctx).Query(loadQ, pg.Strings(txHashes), pg.Uint32s(indexes))
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
			&o.Value,
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
	rows, err = pg.FromContext(ctx).Query(spentQ, pg.Strings(txHashes), pg.Uint32s(indexes))
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
	SELECT o.tx_hash, o.index, o.asset_id, o.amount, o.script, o.metadata
	FROM utxos o LEFT JOIN pool_inputs i USING (tx_hash, index)
	WHERE o.contract_hash = $1 AND o.asset_id = $2 AND i.tx_hash IS NULL AND NOT o.confirmed
`

// LoadPoolUTXOs loads all unspent outputs in the tx pool
// for the given asset and account.
func LoadPoolUTXOs(ctx context.Context, accountID, assetID string) ([]*utxodb.UTXO, error) {
	// TODO(kr): account stuff will split into a separate
	// table and this will become something like
	// LoadPoolUTXOs(context.Context, []bc.Outpoint) []*bc.TxOutput.

	const q = `
		SELECT amount, reserved_until, out.tx_hash, out.index, key_index(addr_index), contract_hash
		FROM utxos out
		LEFT JOIN pool_inputs inp ON ((out.tx_hash, out.index) = (inp.tx_hash, inp.index))
		WHERE account_id=$1 AND asset_id=$2 AND inp.tx_hash IS NULL AND NOT out.confirmed
	`
	rows, err := pg.FromContext(ctx).Query(q, accountID, assetID)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()
	var utxos []*utxodb.UTXO
	for rows.Next() {
		u := &utxodb.UTXO{
			AccountID: accountID,
			AssetID:   assetID,
		}
		var (
			txid      string
			addrIndex []uint32
		)
		var contractHash sql.NullString
		err = rows.Scan(
			&u.Amount,
			&u.ResvExpires,
			&txid,
			&u.Outpoint.Index,
			(*pg.Uint32s)(&addrIndex),
			&contractHash,
		)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		if contractHash.Valid {
			u.ContractHash = contractHash.String
		}
		copy(u.AddrIndex[:], addrIndex)
		h, err := bc.ParseHash(txid)
		if err != nil {
			return nil, errors.Wrap(err, "decode hash")
		}
		u.Outpoint.Hash = h
		u.ResvExpires = u.ResvExpires.UTC()
		utxos = append(utxos, u)
	}
	return utxos, errors.Wrap(rows.Err(), "rows")
}

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

func InsertPoolTx(ctx context.Context, tx *bc.Tx) error {
	const q = `INSERT INTO pool_txs (tx_hash, data) VALUES ($1, $2)`
	_, err := pg.FromContext(ctx).Exec(q, tx.Hash, tx)
	return errors.Wrap(err)
}

func InsertPoolOutputs(ctx context.Context, hash bc.Hash, insert []*Output) error {
	var outs utxoSet
	for i, o := range insert {
		outs.index = append(outs.index, uint32(i))
		outs.assetID = append(outs.assetID, o.AssetID.String())
		outs.amount = append(outs.amount, int64(o.Value))
		outs.accountID = append(outs.accountID, o.AccountID)
		outs.managerNodeID = append(outs.managerNodeID, o.ManagerNodeID)
		outs.aIndex = append(outs.aIndex, toKeyIndex(o.AddrIndex[:]))
		outs.script = append(outs.script, o.Script)

		isPayToContract, contractHash := txscript.TestPayToContract(o.Script)
		if isPayToContract {
			outs.contractHash = append(outs.contractHash, contractHash[:])
		} else {
			outs.contractHash = append(outs.contractHash, nil)
		}
	}

	const q = `
		INSERT INTO utxos (
			tx_hash, pool_tx_hash, index, asset_id, amount,
			account_id, manager_node_id, addr_index,
			script, contract_hash, confirmed
		)
		SELECT
			$1::text,
			$1::text,
			unnest($2::bigint[]),
			unnest($3::text[]),
			unnest($4::bigint[]),
			unnest($5::text[]),
			unnest($6::text[]),
			unnest($7::bigint[]),
			unnest($8::bytea[]),
			unnest($9::bytea[]),
			FALSE
	`
	_, err := pg.FromContext(ctx).Exec(q,
		hash.String(),
		outs.index,
		outs.assetID,
		outs.amount,
		outs.accountID,
		outs.managerNodeID,
		outs.aIndex,
		outs.script,
		outs.contractHash,
	)
	return errors.Wrap(err)
}

// InsertPoolInputs inserts outpoints into pool_inputs.
func InsertPoolInputs(ctx context.Context, outs []bc.Outpoint) error {
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
	_, err := pg.FromContext(ctx).Exec(q, pg.Strings(txHashes), pg.Uint32s(index))
	return errors.Wrap(err)
}

// CountPoolTxs returns the total number of unconfirmed transactions.
func CountPoolTxs(ctx context.Context) (uint64, error) {
	const q = `SELECT count(tx_hash) FROM pool_txs`
	var res uint64
	err := pg.FromContext(ctx).QueryRow(q).Scan(&res)
	return res, errors.Wrap(err)
}

func toKeyIndex(i []uint32) int64 {
	return int64(i[0])<<31 | int64(i[1]&0x7fffffff)
}
