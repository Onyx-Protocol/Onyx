package asset

import (
	"github.com/btcsuite/btcd/btcec"

	"chain/fedchain-sandbox/hdkey"
)

func SignTxTemplate(txTemplate *TxTemplate, priv *hdkey.XKey) error {
	for _, input := range txTemplate.Inputs {
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
