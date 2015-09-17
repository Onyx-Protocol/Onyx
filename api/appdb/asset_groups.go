package appdb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/metrics"
)

// CreateAssetGroup creates a new asset group,
// also adding its xpub to the keys table if necessary.
func CreateAssetGroup(ctx context.Context, appID, label string, xpubs []*Key) (id string, err error) {
	_ = pg.FromContext(ctx).(pg.Tx) // panic if not in a db transaction
	if label == "" {
		return "", ErrBadLabel
	} else if len(xpubs) != 1 {
		// only 1-of-1 supported so far
		return "", ErrBadXPubCount
	}
	for i, xpub := range xpubs {
		if xpub.XPub.IsPrivate() {
			err := errors.WithDetailf(ErrBadXPub, "key number %d", i)
			return "", errors.WithDetail(err, "key is xpriv, not xpub")
		}
	}

	err = upsertKeys(ctx, xpubs...)
	if err != nil {
		return "", errors.Wrap(err, "upsert keys")
	}

	var keyIDs []string
	for _, xpub := range xpubs {
		keyIDs = append(keyIDs, xpub.ID)
	}

	const q = `
		INSERT INTO asset_groups (label, application_id, keyset)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	err = pg.FromContext(ctx).QueryRow(q, label, appID, pg.Strings(keyIDs)).Scan(&id)
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
			(SELECT array_agg(xpub) FROM keys WHERE id=ANY(keyset)),
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

	for _, xpub := range xpubs {
		key, err := NewKey(xpub)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "asset group %v: bad key %v", agID, xpub)
		}
		asset.Keys = append(asset.Keys, key)
	}

	return asset, sigsReq, nil
}
