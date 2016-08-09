package asset_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	. "chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

// TestConflictingTxsInPool tests creating conflicting transactions, and
// ensures that they both make it into the tx pool. Then, when a block
// lands, only one of the txs should be confirmed.
//
// Conflicting txs are created by building a tx template with only a
// source, and then building two different txs with that same source,
// but destinations w/ different addresses.
func TestConflictingTxsInPool(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	info, g, err := bootdb(ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	_, err = issue(ctx, t, info, info.acctA.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	dumpState(ctx, t)
	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	dumpState(ctx, t)

	assetAmount := bc.AssetAmount{
		AssetID: info.asset.AssetID,
		Amount:  10,
	}
	spendAction := assettest.NewAccountSpendAction(assetAmount, info.acctA.ID, nil, nil, nil)
	spendAction.Params.TTL = time.Millisecond
	dest1 := assettest.NewAccountControlAction(assetAmount, info.acctB.ID, nil)

	// Build the first tx
	firstTemplate, err := txbuilder.Build(ctx, nil, []txbuilder.Action{spendAction, dest1}, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	assettest.SignTxTemplate(t, firstTemplate, info.privKeyAccounts)
	tx, err := FinalizeTx(ctx, firstTemplate)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Build the second tx
	secondTemplate, err := txbuilder.Build(ctx, &tx.TxData, nil, []byte("test"))
	secondTemplate.Inputs = firstTemplate.Inputs
	txbuilder.ComputeSigHashes(secondTemplate)
	assettest.SignTxTemplate(t, secondTemplate, info.privKeyAccounts)
	_, err = FinalizeTx(ctx, secondTemplate)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Make a block, which should reject one of the txs.
	dumpState(ctx, t)
	b, err := g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	dumpState(ctx, t)
	if len(b.Transactions) != 1 {
		t.Errorf("got block.Transactions = %#v\n, want exactly one tx", b.Transactions)
	}
}
