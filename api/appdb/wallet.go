package appdb

import (
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
)

// Errors returned by CreateWallet.
// May be wrapped using package chain/errors.
var (
	ErrBadLabel     = errors.New("bad label")
	ErrBadXPubCount = errors.New("bad xpub count")
	ErrXPriv        = errors.New("xpriv given for xpub")
)

// CreateWallet creates a new wallet,
// also adding its xpub to the keys table if necessary.
func CreateWallet(ctx context.Context, appID, label string, xpubs []*Key) (id string, err error) {
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

	const q = `
		INSERT INTO wallets (label, application_id)
		VALUES ($1, $2)
		RETURNING id
	`
	err = pg.FromContext(ctx).QueryRow(q, label, appID).Scan(&id)
	if err != nil {
		return "", errors.Wrap(err, "insert wallet")
	}

	var keyIDs []string
	for _, xpub := range xpubs {
		keyIDs = append(keyIDs, xpub.ID)
	}
	err = createRotation(ctx, id, keyIDs...)
	if err != nil {
		return "", errors.Wrap(err, "create rotation")
	}

	return id, nil
}

func createRotation(ctx context.Context, walletID string, hashes ...string) error {
	const q = `
		WITH new_rotation AS (
			INSERT INTO rotations (wallet_id, keyset)
			VALUES ($1, $2)
			RETURNING id
		)
		UPDATE wallets SET current_rotation=(SELECT id FROM new_rotation)
		WHERE id=$1
	`
	_, err := pg.FromContext(ctx).Exec(q, walletID, pg.Strings(hashes))
	return err
}
