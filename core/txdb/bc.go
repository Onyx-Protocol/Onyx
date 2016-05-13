package txdb

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
)

type Output struct {
	state.Output
	ManagerNodeID string
	AccountID     string
	AddrIndex     [2]uint32
}

func loadOutputs(ctx context.Context, db pg.DB, ps []bc.Outpoint) (map[bc.Outpoint]*state.Output, error) {
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
	outs := make(map[bc.Outpoint]*state.Output)
	err := pg.ForQueryRows(pg.NewContext(ctx, db), q, pg.Strings(txHashes), pg.Uint32s(indexes), func(hash bc.Hash, index uint32, assetID bc.AssetID, amount uint64, script, metadata []byte) {
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
	return outs, err
}
