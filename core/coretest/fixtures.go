// Package coretest provides utilities for testing Chain Core.
package coretest

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/testutil"
)

func CreatePins(ctx context.Context, t testing.TB, s *pin.Store) {
	pins := []string{
		account.PinName,
		account.ExpirePinName,
		account.DeleteSpentsPinName,
		asset.PinName,
		query.TxPinName,
	}
	for _, p := range pins {
		err := s.CreatePin(ctx, p, 0)
		if err != nil {
			testutil.FatalErr(t, err)
		}
	}
}

func CreateAccount(ctx context.Context, t testing.TB, accounts *account.Manager, alias string, tags map[string]interface{}) string {
	keys := []chainkd.XPub{testutil.TestXPub}
	acc, err := accounts.Create(ctx, keys, 1, alias, tags, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return acc.ID
}

func CreateAsset(ctx context.Context, t testing.TB, assets *asset.Registry, def map[string]interface{}, alias string, tags map[string]interface{}) bc.AssetID {
	keys := []chainkd.XPub{testutil.TestXPub}
	asset, err := assets.Define(ctx, keys, 1, def, alias, tags, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return asset.AssetID
}

func IssueAssets(ctx context.Context, t testing.TB, c *protocol.Chain, s txbuilder.Submitter, assets *asset.Registry, accounts *account.Manager, assetID bc.AssetID, amount uint64, accountID string) (*legacy.TxOutput, *bc.Output, bc.Hash) {
	assetAmount := bc.AssetAmount{AssetId: &assetID, Amount: amount}

	tpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{
		assets.NewIssueAction(assetAmount, nil), // does not support reference data
		accounts.NewControlAction(assetAmount, accountID, nil),
	}, time.Now().Add(time.Hour))
	if err != nil {
		testutil.FatalErr(t, err)
	}

	SignTxTemplate(t, ctx, tpl, &testutil.TestXPrv)

	err = txbuilder.FinalizeTx(ctx, c, s, tpl.Transaction)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	outID0 := tpl.Transaction.OutputID(0)
	out0 := tpl.Transaction.Entries[*outID0].(*bc.Output)
	return tpl.Transaction.Outputs[0], out0, *outID0
}

func Transfer(ctx context.Context, t testing.TB, c *protocol.Chain, s txbuilder.Submitter, actions []txbuilder.Action) *legacy.Tx {
	template, err := txbuilder.Build(ctx, nil, actions, time.Now().Add(time.Hour))
	if err != nil {
		testutil.FatalErr(t, err)
	}

	SignTxTemplate(t, ctx, template, &testutil.TestXPrv)

	tx := legacy.NewTx(template.Transaction.TxData)
	err = txbuilder.FinalizeTx(ctx, c, s, tx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return tx
}

func SignTxTemplate(t testing.TB, ctx context.Context, template *txbuilder.Template, priv *chainkd.XPrv) {
	if priv == nil {
		priv = &testutil.TestXPrv
	}
	err := txbuilder.Sign(ctx, template, []chainkd.XPub{priv.XPub()}, func(_ context.Context, _ chainkd.XPub, path [][]byte, data [32]byte) ([]byte, error) {
		derived := priv.Derive(path)
		return derived.Sign(data[:]), nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
