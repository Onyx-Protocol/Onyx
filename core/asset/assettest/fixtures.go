package assettest

import (
	"context"
	"testing"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/txbuilder"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/testutil"
)

func CreateAccountFixture(ctx context.Context, t testing.TB, keys []string, quorum int, alias string, tags map[string]interface{}) string {
	if keys == nil {
		keys = []string{testutil.TestXPub.String()}
	}
	if quorum == 0 {
		quorum = len(keys)
	}
	acc, err := account.Create(ctx, keys, quorum, alias, tags, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return acc.ID
}

func CreateAssetFixture(ctx context.Context, t testing.TB, keys []string, quorum int, def map[string]interface{}, alias string, tags map[string]interface{}) bc.AssetID {
	if len(keys) == 0 {
		keys = []string{testutil.TestXPub.String()}
	}

	if quorum == 0 {
		quorum = len(keys)
	}
	var initialBlockHash bc.Hash

	asset, err := asset.Define(ctx, keys, quorum, def, initialBlockHash, alias, tags, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return asset.AssetID
}

func IssueAssetsFixture(ctx context.Context, t testing.TB, c *protocol.Chain, assetID bc.AssetID, amount uint64, accountID string) state.Output {
	if accountID == "" {
		accountID = CreateAccountFixture(ctx, t, nil, 0, "", nil)
	}
	dest := account.NewControlAction(bc.AssetAmount{AssetID: assetID, Amount: amount}, accountID, nil)

	assetAmount := bc.AssetAmount{AssetID: assetID, Amount: amount}

	src := asset.NewIssueAction(assetAmount, nil) // does not support reference data
	tpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{dest, src})
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
	template, err := txbuilder.Build(ctx, nil, actions)
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
