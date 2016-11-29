package chainkd

import (
	"io"

	"chain-stealth/crypto/ed25519"
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

func DeriveXPubs(xpubs []XPub, path [][]byte) []XPub {
	res := make([]XPub, 0, len(xpubs))
	for _, xpub := range xpubs {
		d := xpub.Derive(path)
		res = append(res, d)
	}
	return res
}
