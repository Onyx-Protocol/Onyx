// Package coretest provides utilities for testing Chain Core.
package coretest

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/pin"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/testutil"
)

func CreatePins(ctx context.Context, t testing.TB, s *pin.Store) {
	pins := []string{account.PinName, asset.PinName, "tx"} // "tx" avoids circular dependency on query
	for _, p := range pins {
		err := s.CreatePin(ctx, p, 0)
		if err != nil {
			testutil.FatalErr(t, err)
		}
	}
}

func CreateAccount(ctx context.Context, t testing.TB, accounts *account.Manager, alias string, tags map[string]interface{}) string {
	keys := []string{testutil.TestXPub.String()}
	acc, err := accounts.Create(ctx, keys, 1, alias, tags, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return acc.ID
}

func CreateAsset(ctx context.Context, t testing.TB, assets *asset.Registry, def map[string]interface{}, alias string, tags map[string]interface{}) bc.AssetID {
	keys := []string{testutil.TestXPub.String()}
	asset, err := assets.Define(ctx, keys, 1, def, alias, tags, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return asset.AssetID
}

func IssueAssets(ctx context.Context, t testing.TB, c *protocol.Chain, assets *asset.Registry, accounts *account.Manager, assetID bc.AssetID, amount uint64, accountID string) state.Output {
	assetAmount := bc.AssetAmount{AssetID: assetID, Amount: amount}

	tpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{
		assets.NewIssueAction(assetAmount, nil), // does not support reference data
		accounts.NewControlAction(bc.AssetAmount{AssetID: assetID, Amount: amount}, accountID, nil),
	}, time.Now().Add(time.Minute))
	if err != nil {
		testutil.FatalErr(t, err)
	}

	SignTxTemplate(t, ctx, tpl, &testutil.TestXPrv)

	err = txbuilder.FinalizeTx(ctx, c, bc.NewTx(*tpl.Transaction))
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return state.Output{
		Outpoint: bc.Outpoint{Hash: tpl.Transaction.Hash(), Index: 0},
		TxOutput: *tpl.Transaction.Outputs[0],
	}
}

func Transfer(ctx context.Context, t testing.TB, c *protocol.Chain, actions []txbuilder.Action) *bc.Tx {
	template, err := txbuilder.Build(ctx, nil, actions, time.Now().Add(time.Minute))
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	SignTxTemplate(t, ctx, template, &testutil.TestXPrv)

	tx := bc.NewTx(*template.Transaction)
	err = txbuilder.FinalizeTx(ctx, c, tx)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	return tx
}

func SignTxTemplate(t testing.TB, ctx context.Context, template *txbuilder.Template, priv *chainkd.XPrv) {
	if priv == nil {
		priv = &testutil.TestXPrv
	}
	err := txbuilder.Sign(ctx, template, []string{priv.XPub().String()}, func(_ context.Context, _ string, path [][]byte, data [32]byte) ([]byte, error) {
		derived := priv.Derive(path)
		return derived.Sign(data[:]), nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
