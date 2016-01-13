package testutil

import (
	"github.com/btcsuite/btcutil/hdkeychain"

	"chain/fedchain-sandbox/hdkey"
)

var (
	TestXPub, TestXPrv *hdkey.XKey
)

func init() {
	seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
	if err != nil {
		panic(err)
	}
	xprv, err := hdkeychain.NewMaster(seed)
	if err != nil {
		panic(err)
	}
	xpub, err := xprv.Neuter()
	if err != nil {
		panic(err)
	}
	TestXPub = &hdkey.XKey{ExtendedKey: *xpub}
	TestXPrv = &hdkey.XKey{ExtendedKey: *xprv}
}
