package txdb

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
)

type view struct {
	isPool bool
	outs   map[bc.Outpoint]*state.Output
	err    *error

	// TODO(kr): preload circulation and delete this field
	db pg.DB // for circulation
}

func newPoolViewForPrevouts(ctx context.Context, db *sql.DB, txs []*bc.Tx) (state.ViewReader, error) {
	var p []bc.Outpoint
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			p = append(p, in.Previous)
		}
	}

	dbtx, err := db.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	ctx = pg.NewContext(ctx, dbtx)
	defer dbtx.Rollback(ctx)

	v, err := newPoolView(ctx, dbtx, db, p)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return v, nil
}

// newPoolView returns a new state view on the pool
// of unconfirmed transactions.
// It loads the outpoints identified in p;
// all other outputs will be omitted from the view.
// Parameter db is used only for Circulation.
func newPoolView(ctx context.Context, dbtx *sql.Tx, db *sql.DB, p []bc.Outpoint) (state.ViewReader, error) {
	outs, err := loadPoolOutputs(ctx, dbtx, p)
	if err != nil {
		return nil, err
	}
	var errbuf error
	result := &view{
		isPool: true,
		outs:   outs,
		err:    &errbuf,
		db:     db, // TODO(kr): preload circulation and delete this field
	}
	return result, nil
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

func (v *view) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	if *v.err != nil {
		return nil
	}
	return v.outs[p]
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
