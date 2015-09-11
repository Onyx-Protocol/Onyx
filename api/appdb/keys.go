package appdb

import (
	"encoding/hex"
	"sort"

	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/hdkey"
	"chain/strings"
)

// ErrMissingKeys is returned by loadKeys
var ErrMissingKeys = errors.New("could not load all keys")

// Key is an xpub and its hash.
type Key struct {
	ID   string     `json:"key_id"` // ID is the hash of the XPub
	XPub hdkey.XKey `json:"xpub"`
}

// NewKey returns a Key object given its data.
func NewKey(pubstr string) (*Key, error) {
	extkey, err := hdkeychain.NewKeyFromString(pubstr)
	if err != nil {
		return nil, err
	}
	k := &Key{
		ID:   HashXPub(pubstr),
		XPub: hdkey.XKey{ExtendedKey: *extkey},
	}
	return k, nil
}

// GetKeys gets the given keys from the db,
// using id to identify them.
func GetKeys(ctx context.Context, ids []string) (ks []*Key, err error) {
	for _, s := range ids {
		ks = append(ks, &Key{ID: s})
	}
	return ks, loadKeys(ctx, ks...)
}

// loadKeys loads the given keys from the db,
// using id to identify them.
func loadKeys(ctx context.Context, keys ...*Key) error {
	var a []string
	for _, k := range keys {
		if k.ID == "" {
			return errors.New("cannot load key without hash")
		}
		a = append(a, k.ID)
	}
	sort.Strings(a)
	a = strings.Uniq(a)
	const q = `SELECT id, xpub FROM keys WHERE id=ANY($1)`
	rows, err := pg.FromContext(ctx).Query(q, pg.Strings(a))
	if err != nil {
		return err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, xpub string
		err := rows.Scan(&id, &xpub)
		if err != nil {
			return err
		}
		for _, k := range keys {
			if k.ID == id {
				k1, err := NewKey(xpub)
				if err != nil {
					return err
				}
				*k = *k1
				n++
			}
		}
	}
	err = rows.Err()
	if err == nil && n != len(keys) {
		err = ErrMissingKeys
	}
	return err
}

// upsertKeys inserts the client keys in keys
// that aren't already in the database.
func upsertKeys(ctx context.Context, keys ...*Key) error {
	const q = `
		WITH nk AS (
				SELECT unnest($1::text[]) id, unnest($2::text[]) xpub
		)
		INSERT INTO keys (id, xpub)
		(
				SELECT id, xpub FROM nk
				WHERE nk.id NOT IN (SELECT id FROM keys)
		)
	`
	var id, xpub pg.Strings
	for _, key := range keys {
		id = append(id, key.ID)
		xpub = append(xpub, key.XPub.String())
	}
	_, err := pg.FromContext(ctx).Exec(q, id, xpub)
	if err != nil {
		return errors.Wrap(err, "insert keys")
	}
	return err
}

// HashXPub is the hex encoded Hash160 of an xpub string
func HashXPub(keystr string) string {
	hash := btcutil.Hash160([]byte(keystr))
	return hex.EncodeToString(hash)
}

func keyIDs(keys []*Key) []string {
	var a []string
	for _, k := range keys {
		a = append(a, k.ID)
	}
	return a
}
