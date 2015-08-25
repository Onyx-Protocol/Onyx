package wallets

import (
	"chain/strings"
	"encoding/hex"
	"errors"
	"sort"

	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/chain-engineering/pg"
)

// Key is a wrapper around an extended key, containing metadata
type Key struct {
	ID       string // ID is the hash of the XPub
	XPub     *hdkeychain.ExtendedKey
	XPrivEnc string
	Type     string // either chain or client
}

// HashXPub is the hex encoded Hash160 of an xpub string
func HashXPub(keystr string) string {
	hash := btcutil.Hash160([]byte(keystr))
	return hex.EncodeToString(hash)
}

// NewKey returns a Key object given its data
func NewKey(pubstr, privEnc, keytype string) (*Key, error) {
	extkey, err := hdkeychain.NewKeyFromString(pubstr)
	if err != nil {
		return nil, err
	}
	k := &Key{
		ID:       HashXPub(pubstr),
		XPub:     extkey,
		XPrivEnc: privEnc,
		Type:     keytype,
	}
	return k, nil
}

// getKeys gets the given keys from the db,
// using id to identify them.
func getKeys(ids []string) (ks []*Key, err error) {
	for _, s := range ids {
		ks = append(ks, &Key{ID: s})
	}
	return ks, loadKeys(ks...)
}

// loadKeys loads the given keys from the db,
// using id to identify them.
func loadKeys(keys ...*Key) error {
	var a []string
	for _, k := range keys {
		if k.ID == "" {
			return errors.New("cannot load key without hash")
		}
		a = append(a, k.ID)
	}
	sort.Strings(a)
	a = strings.Uniq(a)
	const q = `
		SELECT id, type, xpub, COALESCE(enc_xpriv, '')
		FROM keys
		WHERE id=ANY($1)
	`
	rows, err := db.Query(q, pg.SliceString(a))
	if err != nil {
		return err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, keyType, xpub, xprv string
		err := rows.Scan(&id, &keyType, &xpub, &xprv)
		if err != nil {
			return err
		}
		for _, k := range keys {
			if k.ID == id {
				k1, err := NewKey(xpub, xprv, keyType)
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
		err = errors.New("could not load all keys")
	}
	return err
}
