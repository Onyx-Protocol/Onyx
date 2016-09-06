package assettest

import (
	"context"
	"testing"

	"chain/core/txbuilder"
	"chain/crypto/ed25519/hd25519"
	"chain/testutil"
)

func SignTxTemplate(t testing.TB, template *txbuilder.Template, priv *hd25519.XPrv) {
	if priv == nil {
		priv = testutil.TestXPrv
	}
	for i, input := range template.Inputs {
		for _, c := range input.WitnessComponents {
			err := c.Sign(nil, template, i, func(_ context.Context, _ string, path []uint32, data [32]byte) ([]byte, error) {
				derived := priv.Derive(path)
				return derived.Sign(data[:]), nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
