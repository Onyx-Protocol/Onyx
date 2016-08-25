package core

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset/assettest"
	"chain/core/txbuilder"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/testutil"
)

func TestLocalAccountTransfer(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	c, g, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	acc, err := account.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	assetID := assettest.CreateAssetFixture(ctx, t, nil, 1, nil, "", nil)
	assetAmt := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}

	sources := txbuilder.Action(assettest.NewIssueAction(assetAmt, nil))
	dests := assettest.NewAccountControlAction(assetAmt, acc.ID, nil)

	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, nil)
	if err != nil {
		t.Fatal(err)
	}
	assettest.SignTxTemplate(t, tmpl, testutil.TestXPrv)

	// Submit the transaction but w/o waiting long for confirmation.
	// The outputs should be indexed because the transaction template
	// indicates that the transaction is completely local to this Core.
	_, _ = submitSingle(ctx, c, submitSingleArg{tpl: tmpl, wait: time.Millisecond})

	// Add a new source, spending the change output produced above.
	sources = assettest.NewAccountSpendAction(assetAmt, acc.ID, nil, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, nil)
	if err != nil {
		t.Fatal(err)
	}

	assettest.SignTxTemplate(t, tmpl, testutil.TestXPrv)
	_, err = txbuilder.FinalizeTx(ctx, c, tmpl)
	if err != nil {
		t.Fatal(err)
	}

	b, err := g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Transactions) != 2 {
		t.Errorf("len(b.Transactions) = %d, want 2", len(b.Transactions))
	}
}
