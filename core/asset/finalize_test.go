package asset_test

import (
	"reflect"
	"testing"

	. "chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/txdb"
	"chain/cos/bc"
	"chain/cos/state"
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
