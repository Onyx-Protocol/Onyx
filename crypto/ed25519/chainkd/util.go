package chainkd

import (
	"bytes"
	"io"
	"sort"

	"chain/crypto/ed25519"
)

// Utility functions

func NewXKeys(r io.Reader) (xprv XPrv, xpub XPub, err error) {
	xprv, err = NewXPrv(r)
	if err != nil {
		return
	}
	return xprv, xprv.XPub(), nil
}

func XPubKeys(xpubs []XPub) []ed25519.PublicKey {
	res := make([]ed25519.PublicKey, 0, len(xpubs))
	for _, xpub := range xpubs {
		res = append(res, xpub.PublicKey())
	}
	return res
}

func DeriveXPubs(xpubs []XPub, path []uint32) []XPub {
	res := make([]XPub, 0, len(xpubs))
	for _, xpub := range xpubs {
		d := xpub.Derive(path)
		res = append(res, d)
	}
	sort.Sort(byKey(res))
	return res
}

type byKey []XPub

func (a byKey) Len() int      { return len(a) }
func (a byKey) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byKey) Less(i, j int) bool {
	return bytes.Compare(a[i].Bytes(), a[j].Bytes()) < 0
}
