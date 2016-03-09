package txdb

import (
	"golang.org/x/net/context"

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
		FROM utxos_status
		WHERE confirmed
		    AND (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	rows, err := pg.Query(ctx, q, pg.Strings(txHashes), pg.Uint32s(indexes))
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
			&o.Amount,
			&o.Script,
			&o.Metadata,
		)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		outs[o.Outpoint] = o
	}
	return outs, errors.Wrap(rows.Err(), "end row scan loop")
}
