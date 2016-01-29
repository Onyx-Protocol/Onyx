package appdb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/metrics"
	"chain/net/trace/span"
)

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

// AssetAmount is a composite representation of a sum of an asset.
// Confirmed reflects the amount of the asset present in blocks.
// Total includes amounts from unconfirmed transactions.
type AssetAmount struct {
	Confirmed uint64 `json:"confirmed"`
	Total     uint64 `json:"total"`
}

// AssetResponse is a JSON-serializable version of Asset, intended for use in
// API responses.
type AssetResponse struct {
	ID         bc.AssetID         `json:"id"`
	Label      string             `json:"label"`
	Definition chainjson.HexBytes `json:"definition"`
	Issued     AssetAmount        `json:"issued"`

	// Deprecated in its current form, which is equivalent to Issued.Total
	Circulation uint64 `json:"circulation"`
}

// AssetOwner indicates either an account or a manager node.
type AssetOwner int

// Valid values for AssetOwner.
const (
	OwnerAccount AssetOwner = iota
	OwnerManagerNode
)

// AssetByID loads an asset from the database using its ID. If an asset has
// been archived, this function will return ErrArchived.
func AssetByID(ctx context.Context, hash bc.AssetID) (*Asset, error) {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		SELECT assets.keyset, redeem_script, issuer_node_id,
			key_index(issuer_nodes.key_index), key_index(assets.key_index), definition, assets.archived
		FROM assets
		INNER JOIN issuer_nodes ON issuer_nodes.id=assets.issuer_node_id
		WHERE assets.id=$1
	`
	var (
		xpubs    []string
		archived bool
		a        = &Asset{Hash: hash}
	)

	err := pg.FromContext(ctx).QueryRow(ctx, q, hash.String()).Scan(
		(*pg.Strings)(&xpubs),
		&a.RedeemScript,
		&a.IssuerNodeID,
		(*pg.Uint32s)(&a.INIndex),
		(*pg.Uint32s)(&a.AIndex),
		&a.Definition,
		&archived,
	)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	if archived {
		err = ErrArchived
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
		SELECT id, label, t.confirmed, (t.confirmed + t.pool), definition, sort_id
		FROM assets
		JOIN issuance_totals t ON (asset_id=assets.id)
		WHERE issuer_node_id = $1 AND ($2='' OR sort_id<$2) AND NOT archived
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
		err := rows.Scan(
			&a.ID,
			&a.Label,
			&a.Issued.Confirmed,
			&a.Issued.Total,
			(*[]byte)(&a.Definition),
			&last,
		)
		if err != nil {
			return nil, "", errors.Wrap(err, "row scan")
		}
		a.Circulation = a.Issued.Total // populate deprecated field
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
		SELECT id, label, t.confirmed, (t.confirmed + t.pool), definition
		FROM assets
		JOIN issuance_totals t ON (asset_id=assets.id)
		WHERE id=$1
	`
	a := new(AssetResponse)

	err := pg.FromContext(ctx).QueryRow(ctx, q, assetID).Scan(
		&a.ID,
		&a.Label,
		&a.Issued.Confirmed,
		&a.Issued.Total,
		(*[]byte)(&a.Definition),
	)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	a.Circulation = a.Issued.Total // populate deprecated field
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

// ArchiveAsset marks an asset as archived. Once an asset has been archived, it
// cannot be issued, and it won't show up in listAsset responses.
func ArchiveAsset(ctx context.Context, assetID string) error {
	const q = `UPDATE assets SET archived = true WHERE id = $1`
	db := pg.FromContext(ctx)

	_, err := db.Exec(ctx, q, assetID)
	return errors.Wrap(err, "archive query")
}

// UpdateIssuances modifies the issuance totals of a set of assets by the given
// amounts. The amounts may be negative.
func UpdateIssuances(ctx context.Context, deltas map[bc.AssetID]int64, confirmed bool) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		assetIDs []string
		amounts  []int64
	)
	for aid, amt := range deltas {
		assetIDs = append(assetIDs, aid.String())
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
			id           bc.AssetID
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
		last = res[len(res)-1].AssetID.String()
	}
	return res, last, nil
}
