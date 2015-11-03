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
	"chain/metrics"
)

// loadPoolOutput returns the output named by p, if any.
// If the output is not found, it returns nil (no error).
func loadPoolOutput(ctx context.Context, p bc.Outpoint) (*state.Output, error) {
	hash := p.Hash.String()
	const spentQ = `
		SELECT count(*) FROM pool_inputs
		WHERE tx_hash=$1 AND index=$2
	`
	var n int
	err := pg.FromContext(ctx).QueryRow(spentQ, hash, p.Index).Scan(&n)
	if err != nil {
		return nil, errors.Wrap(err, "input count query")
	}
	if n > 0 {
		o := &state.Output{
			Outpoint: p,
			Spent:    true,
		}
		return o, nil
	}

	const loadQ = `
		SELECT asset_id, amount, script, metadata
		FROM pool_outputs
		WHERE tx_hash=$1 AND index=$2
	`
	o := &state.Output{
		Outpoint: p,
	}
	err = pg.FromContext(ctx).QueryRow(loadQ, p.Hash.String(), p.Index).Scan(
		&o.AssetID,
		&o.Value,
		&o.Script,
		&o.Metadata,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err)
	}
	return o, nil
}

// LoadPoolUTXOs loads all unspent outputs in the tx pool
// for the given asset and account.
func LoadPoolUTXOs(ctx context.Context, accountID, assetID string) ([]*utxodb.UTXO, error) {
	// TODO(kr): account stuff will split into a separate
	// table and this will become something like
	// LoadPoolUTXOs(context.Context, []bc.Outpoint) []*bc.TxOutput.

	const q = `
		SELECT amount, reserved_until, out.tx_hash, out.index, key_index(addr_index)
		FROM pool_outputs out
		LEFT JOIN pool_inputs inp ON ((out.tx_hash, out.index) = (inp.tx_hash, inp.index))
		WHERE account_id=$1 AND asset_id=$2 AND inp.tx_hash IS NULL
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
		err = rows.Scan(
			&u.Amount,
			&u.ResvExpires,
			&txid,
			&u.Outpoint.Index,
			(*pg.Uint32s)(&addrIndex),
		)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
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
	txid          pg.Strings
	index         pg.Uint32s
	assetID       pg.Strings
	amount        pg.Int64s
	addr          pg.Strings
	accountID     pg.Strings
	managerNodeID pg.Strings
	aIndex        pg.Int64s
	script        pg.Byteas
	metadata      pg.Byteas
}

func InsertPoolTx(ctx context.Context, tx *bc.Tx) error {
	const q = `INSERT INTO pool_txs (tx_hash, data) VALUES ($1, $2)`
	hash := tx.Hash()
	_, err := pg.FromContext(ctx).Exec(q, hash.String(), tx)
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
	}

	const q = `
		INSERT INTO pool_outputs (
			tx_hash, index, asset_id, amount,
			account_id, manager_node_id, addr_index,
			script
		)
		SELECT
			$1::text,
			unnest($2::bigint[]),
			unnest($3::text[]),
			unnest($4::bigint[]),
			unnest($5::text[]),
			unnest($6::text[]),
			unnest($7::bigint[]),
			unnest($8::bytea[])
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
	)
	return errors.Wrap(err)
}

// InsertPoolInputs inserts outpoints into pool_inputs.
func InsertPoolInputs(ctx context.Context, outs []bc.Outpoint) error {
	defer metrics.RecordElapsed(time.Now())
	var (
		txid  []string
		index []uint32
	)
	for _, o := range outs {
		txid = append(txid, o.Hash.String())
		index = append(index, o.Index)
	}

	const q = `
		INSERT INTO pool_inputs (tx_hash, index)
		SELECT unnest($1::text[]), unnest($2::bigint[])
	`
	_, err := pg.FromContext(ctx).Exec(q, pg.Strings(txid), pg.Uint32s(index))
	return errors.Wrap(err)
}

func toKeyIndex(i []uint32) int64 {
	return int64(i[0])<<31 | int64(i[1]&0x7fffffff)
}
