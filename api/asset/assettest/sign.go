package assettest

import (
	"testing"

	"github.com/btcsuite/btcd/btcec"

	"chain/api/txbuilder"
	"chain/cos/hdkey"
	"chain/testutil"
)

func SignTxTemplate(t testing.TB, template *txbuilder.Template, priv *hdkey.XKey) {
	for _, input := range template.Inputs {
		for _, sig := range input.Sigs {
			key, err := derive(priv, sig.DerivationPath)
			if err != nil {
				testutil.FatalErr(t, err)
			}
			dat, err := key.Sign(input.SignatureData[:])
			if err != nil {
				testutil.FatalErr(t, err)
			}
			sig.DER = append(dat.Serialize(), 1) // append hashtype SIGHASH_ALL
		}
	}
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
