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

type Balance struct {
	AssetID   string `json:"asset_id"`
	Confirmed int64  `json:"confirmed"`
	Total     int64  `json:"total"`
}

// WalletBalance fetches the balances of assets contained in this wallet.
// It returns a slice of Balances, where each Balance contains an asset ID,
// a confirmed balance, and a total balance. The total and confirmed balances
// are currently the same.
func WalletBalance(ctx context.Context, walletID string) ([]*Balance, error) {
	q := `
		SELECT asset_id, sum(amount)::bigint
		FROM utxos
		WHERE wallet_id=$1
		GROUP BY asset_id
		ORDER BY asset_id
	`
	rows, err := pg.FromContext(ctx).Query(q, walletID)
	if err != nil {
		return nil, errors.Wrap(err, "balance query")
	}
	defer rows.Close()
	var bals []*Balance

	for rows.Next() {
		var (
			assetID string
			bal     int64
		)
		err = rows.Scan(&assetID, &bal)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		b := &Balance{
			AssetID:   assetID,
			Total:     bal,
			Confirmed: bal,
		}
		bals = append(bals, b)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows error")
	}
	return bals, err
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
