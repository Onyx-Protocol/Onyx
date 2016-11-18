package testutil

import (
	"github.com/agl/ed25519"

	"chain/crypto/chainkd"
)

var (
	TestXPub chainkd.XPub
	TestXPrv chainkd.XPrv
	TestPub  *[ed25519.PublicKeySize]byte
	TestPubs []*[ed25519.PublicKeySize]byte
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
	TestPubs = []*[ed25519.PublicKeySize]byte{TestPub}
}
