package asset_test

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	. "chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/cos/state"
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

	// Build a transaction template with a reservation, no destination.
	assetAmount := &bc.AssetAmount{
		AssetID: info.asset.Hash,
		Amount:  10,
	}
	sources := []*txbuilder.Source{
		NewAccountSource(ctx, assetAmount, info.acctA.ID, nil, nil, nil),
	}
	srcTmpl, err := txbuilder.Build(ctx, nil, sources, nil, []byte{}, time.Minute)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Build the first tx
	dest1, err := NewAccountDestination(ctx, assetAmount, info.acctB.ID, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	firstTemplate, err := txbuilder.Build(ctx, srcTmpl, nil, []*txbuilder.Destination{dest1}, []byte{}, time.Minute)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	assettest.SignTxTemplate(t, firstTemplate, info.privKeyAccounts)
	_, err = FinalizeTx(ctx, firstTemplate)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Build the second tx
	dest2, err := NewAccountDestination(ctx, assetAmount, info.acctB.ID, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	secondTemplate, err := txbuilder.Build(ctx, srcTmpl, nil, []*txbuilder.Destination{dest2}, []byte{}, time.Minute)
	if err != nil {
		testutil.FatalErr(t, err)
	}
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

func TestLoadAccountInfo(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	acc := assettest.CreateAccountFixture(ctx, t, nil, 0)
	addr := assettest.CreateAccountControlProgramFixture(ctx, t, acc)

	to1 := bc.NewTxOutput(bc.AssetID{}, 0, addr, nil)
	to2 := bc.NewTxOutput(bc.AssetID{}, 0, []byte("notfound"), nil)

	outs := []*state.Output{{
		TxOutput: *to1,
	}, {
		TxOutput: *to2,
	}}

	got, err := LoadAccountInfo(ctx, outs)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !reflect.DeepEqual(got[0].AccountID, acc) {
		t.Errorf("got = %+v want %+v", got[0].AccountID, acc)
	}

	_ = got
}

func TestDeleteUTXOs(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	_, _, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	asset := assettest.CreateAssetFixture(ctx, t, "", "", "")
	out := assettest.IssueAssetsFixture(ctx, t, asset, 1, "")

	block := &bc.Block{Transactions: []*bc.Tx{
		bc.NewTx(bc.TxData{
			Inputs: []*bc.TxInput{
				bc.NewSpendInput(out.Hash, out.Index, nil, asset, 1, nil, nil),
			},
		}),
	}}
	AddBlock(ctx, block) // actually addBlock; see export_test.go (ugh)

	var n int
	err = pg.QueryRow(ctx, `SELECT count(*) FROM account_utxos`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("count(account_utxos) = %d want 0", n)
	}
}
