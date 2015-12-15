package appdb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/metrics"
	"chain/net/trace/span"
)

// ErrBadAsset is an error that means the string
// used as an asset id was not a valid base58 id.
var ErrBadAsset = errors.New("invalid asset")

// Asset represents an asset type in the blockchain.
// It is made up of extended keys, and paths (indexes) within those keys.
type Asset struct {
	Hash            bc.AssetID // the raw Asset ID
	IssuerNodeID    string
	Label           string
	Keys            []*hdkey.XKey
	INIndex, AIndex []uint32
	RedeemScript    []byte
	IssuanceScript  []byte
	Definition      []byte
}

// AssetCirculation is a JSON-serializeable representation of the total
// quantity of issuances of a particular asset. Confirmed represents the amount
// issued in valid blocks, while total includes issuances from unconfirmed
// transactions in the tx pool.
type AssetCirculation struct {
	Confirmed uint64 `json:"confirmed"`
	Total     uint64 `json:"total"`
}

// AssetResponse is a JSON-serializable version of Asset, intended for use in
// API responses.
type AssetResponse struct {
	ID          string           `json:"id"`
	Label       string           `json:"label"`
	Circulation AssetCirculation `json:"circulation"`
}

// AssetOwner indicates either an account or a manager node.
type AssetOwner int

// Valid values for AssetOwner.
const (
	OwnerAccount AssetOwner = iota
	OwnerManagerNode
)

// AssetByID loads an asset from the database using its ID.
func AssetByID(ctx context.Context, hash bc.AssetID) (*Asset, error) {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		SELECT assets.keyset, redeem_script, issuer_node_id,
			key_index(issuer_nodes.key_index), key_index(assets.key_index), definition
		FROM assets
		INNER JOIN issuer_nodes ON issuer_nodes.id=assets.issuer_node_id
		WHERE assets.id=$1
	`
	var (
		xpubs []string
		a     = &Asset{Hash: hash}
	)

	err := pg.FromContext(ctx).QueryRow(ctx, q, hash.String()).Scan(
		(*pg.Strings)(&xpubs),
		&a.RedeemScript,
		&a.IssuerNodeID,
		(*pg.Uint32s)(&a.INIndex),
		(*pg.Uint32s)(&a.AIndex),
		&a.Definition,
	)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	if err != nil {
		return nil, errors.WithDetailf(err, "asset id=%v", hash.String())
	}

	a.Keys, err = stringsToKeys(xpubs)
	if err != nil {
		return nil, errors.Wrap(err, "parsing keys")
	}

	return a, nil
}

// InsertAsset adds the asset to the database
func InsertAsset(ctx context.Context, asset *Asset) error {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		WITH newasset AS (
			INSERT INTO assets (id, issuer_node_id, key_index, keyset, redeem_script, issuance_script, label, definition)
			VALUES($1, $2, to_key_index($3), $4, $5, $6, $7, $8)
			RETURNING id
		)
		INSERT INTO issuance_totals (asset_id) TABLE newasset;
	`

	_, err := pg.FromContext(ctx).Exec(ctx, q,
		asset.Hash.String(),
		asset.IssuerNodeID,
		pg.Uint32s(asset.AIndex),
		pg.Strings(keysToStrings(asset.Keys)),
		asset.RedeemScript,
		asset.IssuanceScript,
		asset.Label,
		asset.Definition,
	)
	return err
}

// ListAssets returns a paginated list of AssetResponses
// belonging to the given issuer node, along with a sortable id
// for last asset, used to retrieve the next page.
func ListAssets(ctx context.Context, inodeID string, prev string, limit int) ([]*AssetResponse, string, error) {
	q := `
		SELECT id, label, t.confirmed, (t.confirmed + t.pool), sort_id
		FROM assets
		JOIN issuance_totals t ON (asset_id=assets.id)
		WHERE issuer_node_id = $1 AND ($2='' OR sort_id<$2)
		ORDER BY sort_id DESC
		LIMIT $3
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, inodeID, prev, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var (
		assets []*AssetResponse
		last   string
	)
	for rows.Next() {
		a := new(AssetResponse)
		err := rows.Scan(&a.ID, &a.Label, &a.Circulation.Confirmed, &a.Circulation.Total, &last)
		if err != nil {
			return nil, "", errors.Wrap(err, "row scan")
		}
		assets = append(assets, a)
	}

	if err := rows.Err(); err != nil {
		return nil, "", errors.Wrap(err, "end row scan loop")
	}

	return assets, last, nil
}

// GetAsset returns an AssetResponse for the given asset id.
func GetAsset(ctx context.Context, assetID string) (*AssetResponse, error) {
	const q = `
		SELECT id, label, t.confirmed, (t.confirmed + t.pool)
		FROM assets
		JOIN issuance_totals t ON (asset_id=assets.id)
		WHERE id=$1
	`
	a := new(AssetResponse)

	err := pg.FromContext(ctx).QueryRow(ctx, q, assetID).Scan(
		&a.ID, &a.Label, &a.Circulation.Confirmed, &a.Circulation.Total,
	)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	return a, errors.WithDetailf(err, "asset id: %s", assetID)
}

// UpdateAsset updates the label of an asset.
func UpdateAsset(ctx context.Context, assetID string, label *string) error {
	if label == nil {
		return nil
	}
	const q = `UPDATE assets SET label = $2 WHERE id = $1`
	db := pg.FromContext(ctx)
	_, err := db.Exec(ctx, q, assetID, *label)
	return errors.Wrap(err, "update query")
}

// DeleteAsset deletes the asset but only if none of it has been issued.
func DeleteAsset(ctx context.Context, assetID string) error {
	const q = `
		WITH deleted AS (
			DELETE FROM issuance_totals
			WHERE asset_id=$1 AND confirmed=0 AND pool=0
			RETURNING asset_id
		)
		DELETE FROM assets WHERE id IN (TABLE deleted)
	`
	db := pg.FromContext(ctx)
	result, err := db.Exec(ctx, q, assetID)
	if err != nil {
		return errors.Wrap(err, "delete query")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "delete query")
	}
	if rowsAffected == 0 {
		// Distinguish between the asset-not-found case and the
		// assets-issued case.
		const q2 = `SELECT issued FROM assets WHERE id = $1`
		var issued int64
		err = db.QueryRow(ctx, q2, assetID).Scan(&issued)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.WithDetailf(pg.ErrUserInputNotFound, "asset id=%v", assetID)
			}
			return errors.Wrap(err, "delete query")
		}
		if issued != 0 {
			return errors.WithDetailf(ErrCannotDelete, "asset id=%v", assetID)
		}
		// Unexpected error. Could be a race condition where someone else
		// deleted the asset first.
		return errors.New("could not delete asset")
	}
	return nil
}

// UpdateIssuances modifies the issuance totals of a set of assets by the given
// amounts. The amounts may be negative.
func UpdateIssuances(ctx context.Context, deltas map[string]int64, confirmed bool) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		assetIDs []string
		amounts  []int64
	)
	for aid, amt := range deltas {
		assetIDs = append(assetIDs, aid)
		amounts = append(amounts, amt)
	}

	column := "pool"
	if confirmed {
		column = "confirmed"
	}

	q := `
		UPDATE issuance_totals
		SET ` + column + ` = ` + column + ` + updates.amount
		FROM (
			SELECT
				unnest($1::text[]) AS asset_id,
				unnest($2::bigint[]) AS amount
		) AS updates
		WHERE issuance_totals.asset_id = updates.asset_id
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, pg.Strings(assetIDs), pg.Int64s(amounts))
	return errors.Wrap(err)
}

// AssetBalQuery is a parameter struct passed in to AssetBalance
type AssetBalQuery struct {
	Owner   AssetOwner
	OwnerID string
	// Set the following for the full paginated list
	Prev  string
	Limit int

	// EXPERIMENTAL - implemented for Glitterco
	// Set the following for a filtered list of assets
	AssetIDs []string
}

// AssetBalance fetches the balances of assets contained in an asset owner
// (either an account or a manager node).
// It returns a slice of Balances and the last asset ID in the page.
// Each Balance contains an asset ID, a confirmed balance,
// and a total balance. Assets are sorted by their IDs.
func AssetBalance(ctx context.Context, abq *AssetBalQuery) ([]*Balance, string, error) {
	paginating := abq.Limit > 0
	if paginating && len(abq.AssetIDs) > 0 {
		return nil, "", errors.New("cannot set both pagination and asset id filter")
	} else if !paginating && len(abq.AssetIDs) == 0 {
		return nil, "", errors.New("must have limit or asset id filter")
	}

	field := "account_id"
	if abq.Owner == OwnerManagerNode {
		field = "manager_node_id"
	}

	filter := "asset_id=ANY($2)"
	limitQ := ""
	params := []interface{}{abq.OwnerID, pg.Strings(abq.AssetIDs)}
	if paginating {
		filter = "($2='' OR asset_id>$2)"
		limitQ = "LIMIT $3"
		params = []interface{}{abq.OwnerID, abq.Prev, abq.Limit}
	}

	q := `
		SELECT SUM(confirmed), SUM(unconfirmed), asset_id
		FROM (
			SELECT amount AS confirmed, 0 AS unconfirmed, asset_id
				FROM utxos WHERE confirmed AND ` + field + `=$1 AND ` + filter + `
			UNION ALL
			SELECT 0 AS confirmed, amount AS unconfirmed, asset_id
				FROM utxos po WHERE NOT po.confirmed AND ` + field + `=$1 AND ` + filter + `
				AND NOT EXISTS(
					SELECT 1 FROM pool_inputs pi
					WHERE po.tx_hash = pi.tx_hash AND po.index = pi.index
				)
			UNION ALL
			SELECT 0 AS confirmed, amount*-1 AS unconfirmed, asset_id
				FROM utxos u WHERE u.confirmed AND ` + field + `=$1 AND ` + filter + `
				AND EXISTS(
					SELECT 1 FROM pool_inputs pi
					WHERE u.tx_hash = pi.tx_hash AND u.index = pi.index
				)
		) AS bals
		GROUP BY asset_id
		ORDER BY asset_id ASC
	` + limitQ

	rows, err := pg.FromContext(ctx).Query(ctx, q, params...)
	if err != nil {
		return nil, "", errors.Wrap(err, "balance query")
	}
	defer rows.Close()

	var (
		res  []*Balance
		last string
	)
	for rows.Next() {
		var (
			id           string
			conf, unconf int64
		)
		err = rows.Scan(&conf, &unconf, &id)
		if err != nil {
			return nil, "", errors.Wrap(err, "row scan")
		}
		res = append(res, &Balance{
			AssetID:   id,
			Confirmed: conf,
			Total:     conf + unconf,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, "", errors.Wrap(err, "rows end")
	}

	if paginating && len(res) == abq.Limit {
		last = res[len(res)-1].AssetID
	}
	return res, last, nil
}
