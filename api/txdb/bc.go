package txdb

import (
	"database/sql"

	"golang.org/x/net/context"

	"chain/api/utxodb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type Output struct {
	state.Output
	ManagerNodeID string
	AccountID     string
	AddrIndex     [2]uint32
}

func loadOutputs(ctx context.Context, ps []bc.Outpoint) (map[bc.Outpoint]*state.Output, error) {
	var (
		txHashes []string
		indexes  []uint32
	)
	for _, p := range ps {
		txHashes = append(txHashes, p.Hash.String())
		indexes = append(indexes, p.Index)
	}

	const q = `
		SELECT tx_hash, index, asset_id, amount, script, metadata
		FROM utxos
		WHERE confirmed
		    AND (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, pg.Strings(txHashes), pg.Uint32s(indexes))
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer rows.Close()
	outs := make(map[bc.Outpoint]*state.Output)
	for rows.Next() {
		// If the utxo row exists, it is considered unspent. This function does
		// not (and should not) consider spending activity in the tx pool, which
		// is handled by poolView.
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
	return outs, nil
}

const bcUnspentP2COutputQuery = `
	SELECT tx_hash, index, asset_id, amount, script, metadata
	FROM utxos
	WHERE contract_hash = $1 AND asset_id = $2 AND confirmed
`

// LoadUTXOs loads all unspent outputs in the blockchain
// for the given asset and account.
func LoadUTXOs(ctx context.Context, accountID, assetID string) ([]*utxodb.UTXO, error) {
	// TODO(kr): account stuff will split into a separate
	// table and this will become something like
	// LoadUTXOs(context.Context, []bc.Outpoint) []*bc.TxOutput.

	const q = `
		SELECT amount, reserved_until, tx_hash, index, contract_hash, key_index(addr_index)
		FROM utxos
		WHERE account_id=$1 AND asset_id=$2 AND confirmed
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, accountID, assetID)
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
			txid         string
			contractHash sql.NullString
			addrIndex    []uint32
		)
		err = rows.Scan(
			&u.Amount,
			&u.ResvExpires,
			&txid,
			&u.Outpoint.Index,
			&contractHash,
			(*pg.Uint32s)(&addrIndex),
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
	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err())
	}
	return utxos, errors.Wrap(rows.Err(), "rows")
}
