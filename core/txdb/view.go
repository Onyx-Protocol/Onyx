package txdb

import (
	"bytes"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/errors"
)

type view struct {
	isPool bool
	outs   map[bc.Outpoint]*state.Output
	err    *error

	// TODO(kr): preload circulation and delete this field
	db pg.DB // for circulation
}

func newViewForPrevouts(ctx context.Context, db pg.DB, txs []*bc.Tx) (state.ViewReader, error) {
	var p []bc.Outpoint
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			p = append(p, in.Previous)
		}
	}
	return newView(ctx, db, p)
}

// newView returns a new state view on the blockchain.
// It loads the outpoints identified in p;
// all other outputs will be omitted from the view.
func newView(ctx context.Context, db pg.DB, p []bc.Outpoint) (state.ViewReader, error) {
	outs, err := loadOutputs(ctx, db, p)
	if err != nil {
		return nil, err
	}
	var errbuf error
	result := &view{
		outs: outs,
		err:  &errbuf,
		db:   db, // TODO(kr): preload circulation and delete this field
	}
	return result, nil
}

func (v *view) IsUTXO(ctx context.Context, o *state.Output) bool {
	if *v.err != nil {
		return false
	}
	viewOutput := v.outs[o.Outpoint]
	if viewOutput == nil {
		return false
	}
	return viewOutput.AssetAmount == o.AssetAmount && bytes.Equal(viewOutput.Script, o.Script)
}

func (v *view) Circulation(ctx context.Context, assets []bc.AssetID) (map[bc.AssetID]int64, error) {
	return circulation(ctx, v.db, v.isPool, assets)
}

func circulation(ctx context.Context, db pg.DB, inPool bool, assets []bc.AssetID) (map[bc.AssetID]int64, error) {
	const q = `
		SELECT asset_id, (CASE WHEN $2
			THEN confirmed - destroyed_confirmed
			ELSE pool - destroyed_pool
			END)
		FROM issuance_totals WHERE asset_id=ANY($1)
	`
	assetStrs := make([]string, 0, len(assets))
	for _, a := range assets {
		assetStrs = append(assetStrs, a.String())
	}

	circ := make(map[bc.AssetID]int64, len(assets))

	// NOTE(kr): querying circulation here is a bug.
	// See https://github.com/chain-engineering/chain/issues/916.
	err := pg.ForQueryRows(pg.NewContext(ctx, db), q, pg.Strings(assetStrs), !inPool, func(aid bc.AssetID, amt int64) {
		if amt != 0 {
			circ[aid] = amt
		}
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return circ, nil
}

func (v *view) StateRoot(context.Context) (bc.Hash, error) {
	panic("unimplemented")
}
