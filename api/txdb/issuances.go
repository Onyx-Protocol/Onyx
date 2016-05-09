package txdb

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
)

func addIssuances(ctx context.Context, dbtx *sql.Tx, assets map[bc.AssetID]*state.AssetState, confirmed bool) error {
	assetIDs, issued, destroyed := collectIssuedArrays(assets)

	const insertQ = `
		WITH issued AS (
			SELECT * FROM unnest($1::text[], $2::bigint[], $3::bigint[])
				AS t(asset_id, issued, destroyed)
		)
		INSERT INTO issuance_totals(asset_id, confirmed, pool, destroyed_confirmed, destroyed_pool)
		SELECT asset_id,
			(CASE WHEN $4 THEN issued ELSE 0 END),
			(CASE WHEN $4 THEN 0 ELSE issued END),
			(CASE WHEN $4 THEN destroyed ELSE 0 END),
			(CASE WHEN $4 THEN 0 ELSE destroyed END)
		FROM issued
		ON CONFLICT (asset_id) DO UPDATE
		SET confirmed=issuance_totals.confirmed+excluded.confirmed,
			pool=issuance_totals.pool+excluded.pool,
			destroyed_confirmed=issuance_totals.destroyed_confirmed+excluded.destroyed_confirmed,
			destroyed_pool=issuance_totals.destroyed_pool+excluded.destroyed_pool
	`
	_, err := dbtx.Exec(ctx, insertQ, pg.Strings(assetIDs), pg.Uint64s(issued), pg.Uint64s(destroyed), confirmed)
	return errors.Wrap(err, "inserting new issuance_totals")
}

func setIssuances(ctx context.Context, dbtx *sql.Tx, assets map[bc.AssetID]*state.AssetState) error {
	assetIDs, issued, destroyed := collectIssuedArrays(assets)

	const q = `
		WITH issued AS (
			SELECT * FROM unnest($1::text[], $2::bigint[], $3::bigint[]) AS t(asset_id, issued, destroyed)
		)
		UPDATE issuance_totals it
		SET pool=issued, destroyed_pool=destroyed
		FROM issued i WHERE i.asset_id=it.asset_id
	`
	_, err := dbtx.Exec(ctx, q, pg.Strings(assetIDs), pg.Uint64s(issued), pg.Uint64s(destroyed))

	return errors.Wrap(err)
}

// collectIssuedArrays creates 3 parallel slices
// to be used as inputs to sql unnest calls
func collectIssuedArrays(assets map[bc.AssetID]*state.AssetState) (assetIDs []string, issued []uint64, destroyed []uint64) {
	for aid, state := range assets {
		assetIDs = append(assetIDs, aid.String())
		issued = append(issued, state.Issuance)
		destroyed = append(destroyed, state.Destroyed)
	}
	return assetIDs, issued, destroyed
}
