package core

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/txbuilder"
	"chain/database/pg/pgtest"
	chainjson "chain/encoding/json"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestLocalAccountTransfer(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	assets := asset.NewRegistry(db, c)
	accounts := account.NewManager(db, c)
	h := &Handler{Assets: assets, Accounts: accounts, DB: db, Chain: c}

	acc, err := accounts.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	assetID := coretest.CreateAsset(ctx, t, assets, nil, "", nil)
	assetAmt := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}

	sources := txbuilder.Action(assets.NewIssueAction(assetAmt, nil))
	dests := accounts.NewControlAction(assetAmt, acc.ID, nil)

	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)

	// Submit the transaction but w/o waiting long for confirmation.
	// The outputs should be indexed because the transaction template
	// indicates that the transaction is completely local to this Core.
	_, _ = h.submitSingle(ctx, c, submitSingleArg{tpl: tmpl, wait: chainjson.Duration{time.Millisecond}})

	// Add a new source, spending the change output produced above.
	sources = accounts.NewSpendAction(assetAmt, acc.ID, nil, nil, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, bc.NewTx(*tmpl.Transaction))
	if err != nil {
		t.Fatal(err)
	}

	b := prottest.MakeBlock(t, c)
	if len(b.Transactions) != 2 {
		t.Errorf("len(b.Transactions) = %d, want 2", len(b.Transactions))
	}
}
