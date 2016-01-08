package testutil

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/hdkeychain"

	"chain/api/asset"
	"chain/fedchain-sandbox/hdkey"
)

var (
	TestXPub, TestXPrv *hdkey.XKey
)

func SignTx(tx *asset.TxTemplate, priv *hdkey.XKey) error {
	for _, input := range tx.Inputs {
		for _, sig := range input.Sigs {
			key, err := derive(priv, sig.DerivationPath)
			if err != nil {
				return err
			}
			dat, err := key.Sign(input.SignatureData[:])
			if err != nil {
				return err
			}
			sig.DER = append(dat.Serialize(), 1) // append hashtype SIGHASH_ALL
		}
	}
	return nil
}

func derive(xkey *hdkey.XKey, path []uint32) (*btcec.PrivateKey, error) {
	// The only error has a uniformly distributed probability of 1/2^127
	// We've decided to ignore this chance.
	key := &xkey.ExtendedKey
	for _, p := range path {
		key, _ = key.Child(p)
	}
	return key.ECPrivKey()
}

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
