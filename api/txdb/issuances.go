package txdb

import (
	"database/sql"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
)

// issued returns the confirmed and total amounts
// issued of the given asset.
// TODO: export this function and use in other packages
// instead of directly querying
func issued(ctx context.Context, assetID bc.AssetID) (confirmed, total uint64, err error) {
	const q = `
		SELECT confirmed, (confirmed+pool) FROM issuance_totals WHERE asset_id=$1
	`
	err = pg.FromContext(ctx).QueryRow(ctx, q, assetID).Scan(&confirmed, &total)
	if err == sql.ErrNoRows {
		return 0, 0, nil
	}
	return confirmed, total, errors.Wrap(err, "loading issued amounts")
}

func addIssuances(ctx context.Context, issued map[bc.AssetID]uint64, confirmed bool) error {
	assetIDs, amounts := collectIssuedArrays(issued)

	const insertQ = `
		WITH issued AS (
			SELECT * FROM unnest($1::text[], $2::bigint[]) AS t(asset_id, amount)
		)
		INSERT INTO issuance_totals(asset_id, confirmed, pool)
		SELECT asset_id,
			(CASE WHEN $3 THEN amount ELSE 0 END),
			(CASE WHEN $3 THEN 0 ELSE amount END)
		FROM issued
		ON CONFLICT (asset_id) DO UPDATE
		SET confirmed=issuance_totals.confirmed+excluded.confirmed,
			pool=issuance_totals.pool+excluded.pool
	`
	_, err := pg.FromContext(ctx).Exec(ctx, insertQ, pg.Strings(assetIDs), pg.Uint64s(amounts), confirmed)
	return errors.Wrap(err, "inserting new issuance_totals")
}

func removeIssuances(ctx context.Context, issued map[bc.AssetID]uint64) error {
	assetIDs, amounts := collectIssuedArrays(issued)

	const q = `
		WITH issued AS (
			SELECT * FROM unnest($1::text[], $2::bigint[]) AS t(asset_id, amount)
		)
		UPDATE issuance_totals it SET pool=pool-amount
		FROM issued i WHERE it.asset_id=i.asset_id
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, pg.Strings(assetIDs), pg.Uint64s(amounts))

	return errors.Wrap(err)
}

func collectIssuedArrays(issued map[bc.AssetID]uint64) (assetIDs []string, amounts []uint64) {
	for aid, amt := range issued {
		assetIDs = append(assetIDs, aid.String())
		amounts = append(amounts, amt)
	}
	return assetIDs, amounts
}
