package appdb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/metrics"
)

// AssetGroup represents a single asset group. It is intended to be used wth API
// responses.
type AssetGroup struct {
	ID         string `json:"id"`
	Blockchain string `json:"block_chain"`
	Label      string `json:"label"`
}

// CreateAssetGroup creates a new asset group,
// also adding its xpub to the keys table if necessary.
func CreateAssetGroup(ctx context.Context, appID, label string, keys []*hdkey.XKey) (id string, err error) {
	_ = pg.FromContext(ctx).(pg.Tx) // panic if not in a db transaction
	if label == "" {
		return "", ErrBadLabel
	} else if len(keys) != 1 {
		// only 1-of-1 supported so far
		return "", ErrBadXPubCount
	}
	for i, key := range keys {
		if key.IsPrivate() {
			err := errors.WithDetailf(ErrBadXPub, "key number %d", i)
			return "", errors.WithDetail(err, "key is xpriv, not xpub")
		}
	}

	const q = `
		INSERT INTO asset_groups (label, application_id, keyset)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err = pg.FromContext(ctx).QueryRow(q, label, appID, pg.Strings(keysToStrings(keys))).Scan(&id)
	if err != nil {
		return "", errors.Wrap(err, "insert asset group")
	}

	return id, nil
}

// NextAsset returns all data needed
// for creating a new asset. This includes
// all keys, the asset group index, a
// new index for the asset being created,
// and the number of signatures required.
func NextAsset(ctx context.Context, agID string) (asset *Asset, sigsRequired int, err error) {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		UPDATE asset_groups
		SET next_asset_index=next_asset_index+1
		WHERE id=$1
		RETURNING
			keyset,
			key_index(key_index),
			key_index(next_asset_index-1),
			sigs_required
	`
	asset = &Asset{GroupID: agID}
	var (
		xpubs   []string
		sigsReq int
	)
	err = pg.FromContext(ctx).QueryRow(q, agID).Scan(
		(*pg.Strings)(&xpubs),
		(*pg.Uint32s)(&asset.AGIndex),
		(*pg.Uint32s)(&asset.AIndex),
		&sigsReq,
	)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	if err != nil {
		return nil, 0, errors.WithDetailf(err, "asset group %v: get key info", agID)
	}

	asset.Keys, err = stringsToKeys(xpubs)
	if err != nil {
		return nil, 0, errors.Wrap(err, "parsing keys")
	}

	return asset, sigsReq, nil
}

// ListAssetGroups returns a list of AssetGroups belonging to the given
// application.
func ListAssetGroups(ctx context.Context, appID string) ([]*AssetGroup, error) {
	q := `
		SELECT id, block_chain, label
		FROM asset_groups
		WHERE application_id = $1
		ORDER BY created_at
	`
	rows, err := pg.FromContext(ctx).Query(q, appID)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var ags []*AssetGroup
	for rows.Next() {
		ag := new(AssetGroup)
		err := rows.Scan(&ag.ID, &ag.Blockchain, &ag.Label)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		ags = append(ags, ag)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return ags, nil
}

// GetAssetGroup returns basic information about a single asset group.
func GetAssetGroup(ctx context.Context, groupID string) (*AssetGroup, error) {
	var (
		q     = `SELECT label, block_chain FROM asset_groups WHERE id = $1`
		label string
		bc    string
	)
	err := pg.FromContext(ctx).QueryRow(q, groupID).Scan(&label, &bc)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "asset group ID: %v", groupID)
	}
	if err != nil {
		return nil, err
	}

	return &AssetGroup{ID: groupID, Label: label, Blockchain: bc}, nil
}

// UpdateIssuerNode updates the label of an issuer node.
func UpdateIssuerNode(ctx context.Context, inodeID string, label *string) error {
	if label == nil {
		return nil
	}
	const q = `UPDATE asset_groups SET label = $2 WHERE id = $1`
	db := pg.FromContext(ctx)
	_, err := db.Exec(q, inodeID, *label)
	return errors.Wrap(err, "update query")
}
