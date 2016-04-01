package txdb

import (
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type view struct {
	isPool bool
	outs   map[bc.Outpoint]*state.Output
	err    *error
}

func newPoolViewForPrevouts(ctx context.Context, txs []*bc.Tx) (state.ViewReader, error) {
	var p []bc.Outpoint
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			p = append(p, in.Previous)
		}
	}
	return newPoolView(ctx, p)
}

// newPoolView returns a new state view on the pool
// of unconfirmed transactions.
// It loads the outpoints identified in p;
// all other outputs will be omitted from the view.
func newPoolView(ctx context.Context, p []bc.Outpoint) (state.ViewReader, error) {
	outs, err := loadPoolOutputs(ctx, p)
	if err != nil {
		return nil, err
	}
	var errbuf error
	result := &view{
		isPool: true,
		outs:   outs,
		err:    &errbuf,
	}
	return result, nil
}

func newViewForPrevouts(ctx context.Context, txs []*bc.Tx) (state.ViewReader, error) {
	var p []bc.Outpoint
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			p = append(p, in.Previous)
		}
	}
	return newView(ctx, p)
}

// newView returns a new state view on the blockchain.
// It loads the outpoints identified in p;
// all other outputs will be omitted from the view.
func newView(ctx context.Context, p []bc.Outpoint) (state.ViewReader, error) {
	outs, err := loadOutputs(ctx, p)
	if err != nil {
		return nil, err
	}
	var errbuf error
	result := &view{
		outs: outs,
		err:  &errbuf,
	}
	return result, nil
}

func (v *view) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	if *v.err != nil {
		return nil
	}
	return v.outs[p]
}

func (v *view) Circulation(ctx context.Context, assets []bc.AssetID) (map[bc.AssetID]int64, error) {
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

	err := pg.ForQueryRows(ctx, q, pg.Strings(assetStrs), !v.isPool, func(aid bc.AssetID, amt int64) {
		if amt != 0 {
			circ[aid] = amt
		}
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return circ, nil
}
