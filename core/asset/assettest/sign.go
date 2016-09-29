package assettest

import (
	"context"
	"testing"

	"chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/testutil"
)

func SignTxTemplate(t testing.TB, ctx context.Context, template *txbuilder.Template, priv *chainkd.XPrv) {
	if priv == nil {
		priv = &testutil.TestXPrv
	}
	err := txbuilder.Sign(ctx, template, []string{priv.XPub().String()}, func(_ context.Context, _ string, path []uint32, data [32]byte) ([]byte, error) {
		derived := priv.Derive(path)
		return derived.Sign(data[:]), nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
