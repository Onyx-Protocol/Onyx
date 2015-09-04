package appdb

import (
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
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
			return "", errors.WithDetailf(ErrXPriv, "key number %d", i)
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
