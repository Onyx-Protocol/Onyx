package assettest

import (
	"testing"

	"chain/core/txbuilder"
	"chain/crypto/ed25519/hd25519"
)

func SignTxTemplate(t testing.TB, template *txbuilder.Template, priv *hd25519.XPrv) {
	for _, input := range template.Inputs {
		for _, component := range input.SigComponents {
			for _, sig := range component.Signatures {
				derivedSK := priv.Derive(sig.DerivationPath)
				dat := derivedSK.Sign(component.SignatureData[:])
				sig.Bytes = append(dat, 1) // append hashtype SIGHASH_ALL
			}
		}
	}
}
