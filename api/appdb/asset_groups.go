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
	ID          string        `json:"id"`
	Blockchain  string        `json:"block_chain"`
	Label       string        `json:"label"`
	Keys        []*hdkey.XKey `json:"keys,omitempty"`
	SigsReqd    int           `json:"signatures_required,omitempty"`
	PrivateKeys []*hdkey.XKey `json:"private_keys,omitempty"`
}

// InsertAssetGroup adds the asset group to the database
func InsertAssetGroup(ctx context.Context, projID, label string, keys, gennedKeys []*hdkey.XKey) (*AssetGroup, error) {
	_ = pg.FromContext(ctx).(pg.Tx) // panic if not in a db transaction

	const q = `
		INSERT INTO issuer_nodes (label, project_id, keyset, generated_keys)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var id string
	err := pg.FromContext(ctx).QueryRow(q,
		label,
		projID,
		pg.Strings(keysToStrings(keys)),
		pg.Strings(keysToStrings(gennedKeys)),
	).Scan(&id)
	if err != nil {
		return nil, errors.Wrap(err, "insert asset group")
	}

	return &AssetGroup{
		ID:          id,
		Blockchain:  "sandbox",
		Label:       label,
		Keys:        keys,
		SigsReqd:    1,
		PrivateKeys: gennedKeys,
	}, nil
}

// NextAsset returns all data needed
// for creating a new asset. This includes
// all keys, the asset group index, a
// new index for the asset being created,
// and the number of signatures required.
func NextAsset(ctx context.Context, agID string) (asset *Asset, sigsRequired int, err error) {
	defer metrics.RecordElapsed(time.Now())
	const q = `
		UPDATE issuer_nodes
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
// project.
func ListAssetGroups(ctx context.Context, projID string) ([]*AssetGroup, error) {
	q := `
		SELECT id, block_chain, label
		FROM issuer_nodes
		WHERE project_id = $1
		ORDER BY created_at
	`
	rows, err := pg.FromContext(ctx).Query(q, projID)
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
		q       = `SELECT label, block_chain, generated_keys FROM issuer_nodes WHERE id = $1`
		label   string
		bc      string
		keyStrs []string
	)
	err := pg.FromContext(ctx).QueryRow(q, groupID).Scan(&label, &bc, (*pg.Strings)(&keyStrs))
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "asset group ID: %v", groupID)
	}
	if err != nil {
		return nil, err
	}

	keys, err := stringsToKeys(keyStrs)
	if err != nil {
		return nil, errors.Wrap(err, "parsing private keys")
	}

	return &AssetGroup{ID: groupID, Label: label, Blockchain: bc, PrivateKeys: keys}, nil
}

// UpdateIssuerNode updates the label of an issuer node.
func UpdateIssuerNode(ctx context.Context, inodeID string, label *string) error {
	if label == nil {
		return nil
	}
	const q = `UPDATE issuer_nodes SET label = $2 WHERE id = $1`
	db := pg.FromContext(ctx)
	_, err := db.Exec(q, inodeID, *label)
	return errors.Wrap(err, "update query")
}
