package testutil

import (
	"chain-stealth/crypto/ed25519"
	"chain-stealth/crypto/ed25519/chainkd"
)

var (
	TestXPub chainkd.XPub
	TestXPrv chainkd.XPrv
	TestPub  ed25519.PublicKey
	TestPubs []ed25519.PublicKey
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
	TestXPrv, TestXPub, err = chainkd.NewXKeys(zeroReader{})
	if err != nil {
		panic(err)
	}
	TestPub = TestXPub.PublicKey()
	TestPubs = []ed25519.PublicKey{TestPub}
}
