package asset_test

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	. "chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/txdb"
	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestLoadAccountInfo(t *testing.T) {
	ctx := pgtest.NewContext(t)

	mnode := assettest.CreateManagerNodeFixture(ctx, t, "", "", nil, nil)
	acc := assettest.CreateAccountFixture(ctx, t, mnode, "", nil)
	addr := assettest.CreateAddressFixture(ctx, t, acc)

	outs := []*txdb.Output{{
		Output: state.Output{TxOutput: bc.TxOutput{Script: addr.PKScript}},
	}, {
		Output: state.Output{TxOutput: bc.TxOutput{Script: []byte("notfound")}},
	}}

	got, err := LoadAccountInfo(ctx, outs)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want := []*txdb.Output{{
		Output:        state.Output{TxOutput: bc.TxOutput{Script: addr.PKScript}},
		ManagerNodeID: mnode,
		AccountID:     acc,
	}}
	copy(want[0].AddrIndex[:], addr.Index)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got = %+v want %+v", got, want)
	}
}

func TestDeleteUTXOs(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	_, err := assettest.InitializeSigningGenerator(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	asset := assettest.CreateAssetFixture(ctx, t, "", "", "")
	out := assettest.IssueAssetsFixture(ctx, t, asset, 1, "")

	block := &bc.Block{Transactions: []*bc.Tx{
		bc.NewTx(bc.TxData{
			Inputs: []*bc.TxInput{
				{Previous: out.Outpoint},
			},
		}),
	}}
	AddBlock(ctx, block, nil) // actually addBlock; see export_test.go (ugh)

	var n int
	err = pg.QueryRow(ctx, `SELECT count(*) FROM account_utxos`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("count(account_utxos) = %d want 0", n)
	}
}
