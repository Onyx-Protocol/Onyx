package testutil

import (
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/hd25519"
)

var (
	TestXPub *hd25519.XPub
	TestXPrv *hd25519.XPrv
	TestPub  ed25519.PublicKey
	TestPrv  ed25519.PrivateKey
)

type zeroReader struct{}

func (z zeroReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

func init() {
	var err error
	TestXPrv, TestXPub, err = hd25519.NewXKeys(zeroReader{})
	if err != nil {
		panic(err)
	}
	TestPrv = TestXPrv.Key
	TestPub = TestXPub.Key
}

// XPubs parses the serialized xpubs in a.
// If there is a parsing error, it panics.
func XPubs(a ...string) (ks []*hd25519.XPub) {
	for _, s := range a {
		pk, err := hd25519.XPubFromString(s)
		if err != nil {
			panic(err)
		}
		ks = append(ks, pk)
	}
	return ks
}
