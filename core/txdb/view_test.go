package txdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestView(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()
	pgtest.Exec(pg.NewContext(ctx, dbtx), t, `
		INSERT INTO utxos
			(tx_hash, index, asset_id, amount, script, metadata)
		VALUES
			('1000000000000000000000000000000000000000000000000000000000000000', 1, 'A55E710000000000000000000000000000000000000000000000000000000000', 1, 'script-1', 'metadata-1'),
			('2000000000000000000000000000000000000000000000000000000000000000', 2, 'A55E720000000000000000000000000000000000000000000000000000000000', 2, 'script-2', 'metadata-2');

		INSERT INTO blocks_utxos (tx_hash, index)
		VALUES
			('1000000000000000000000000000000000000000000000000000000000000000', 1),
			('2000000000000000000000000000000000000000000000000000000000000000', 2);
	`)

	examples := []struct {
		out   *state.Output
		valid bool
	}{
		{
			valid: true,
			out: &state.Output{
				TxOutput: bc.TxOutput{
					AssetAmount: bc.AssetAmount{AssetID: bc.AssetID(mustParseHash("A55E710000000000000000000000000000000000000000000000000000000000")), Amount: 1},
					Script:      []byte("script-1"),
					Metadata:    []byte("metadata-1"),
				},
				Outpoint: bc.Outpoint{
					Hash:  mustParseHash("1000000000000000000000000000000000000000000000000000000000000000"),
					Index: 1,
				},
			},
		},
		{
			valid: true,
			out: &state.Output{
				TxOutput: bc.TxOutput{
					AssetAmount: bc.AssetAmount{AssetID: bc.AssetID(mustParseHash("A55E720000000000000000000000000000000000000000000000000000000000")), Amount: 2},
					Script:      []byte("script-2"),
					Metadata:    []byte("metadata-2"),
				},
				Outpoint: bc.Outpoint{
					Hash:  mustParseHash("2000000000000000000000000000000000000000000000000000000000000000"),
					Index: 2,
				},
			},
		},
		{
			valid: false,
			out: &state.Output{
				TxOutput: bc.TxOutput{
					AssetAmount: bc.AssetAmount{AssetID: bc.AssetID(mustParseHash("A55E730000000000000000000000000000000000000000000000000000000000")), Amount: 3},
					Script:      []byte("script-3"),
					Metadata:    []byte("metadata-3"),
				},
				Outpoint: bc.Outpoint{
					Hash:  mustParseHash("3000000000000000000000000000000000000000000000000000000000000000"),
					Index: 3,
				},
			},
		},
	}
	for i, ex := range examples {
		t.Log("Example", i)

		v, err := newView(ctx, dbtx, []bc.Outpoint{ex.out.Outpoint})
		if err != nil {
			t.Fatal("unexpected error:", err)
		}

		got := v.IsUTXO(ctx, ex.out)
		if got != ex.valid {
			t.Errorf("output %v, IsUTXO lookup: got=%t, want=%t", ex.out.Outpoint, got, ex.valid)
		}
	}
}

func TestViewForPrevoutsIgnoreIssuance(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()
	txs := []*bc.Tx{bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{{
			Previous: bc.Outpoint{
				Index: 0xffffffff,
			},
		}},
	})}

	v, err := newViewForPrevouts(ctx, dbtx, txs)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	got := len(v.(*view).outs)
	if got != 0 {
		t.Errorf("len(outs) = %d want 0", got)
	}
}

func TestViewCirculation(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()

	assets := []bc.AssetID{{1}, {2}, {3}, {4}, {5}}
	err := addIssuances(ctx, dbtx, map[bc.AssetID]*state.AssetState{
		assets[0]: &state.AssetState{Issuance: 5},
		assets[1]: &state.AssetState{Issuance: 9, Destroyed: 2},
		assets[2]: &state.AssetState{Issuance: 8},
	}, true)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = addIssuances(ctx, dbtx, map[bc.AssetID]*state.AssetState{
		assets[0]: &state.AssetState{Issuance: 5},
		assets[1]: &state.AssetState{Issuance: 9, Destroyed: 2},
		assets[2]: &state.AssetState{Destroyed: 3},
		assets[3]: &state.AssetState{Issuance: 4},
	}, false)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	cases := []struct {
		aids   []bc.AssetID
		isPool bool
		want   map[bc.AssetID]int64
	}{{
		[]bc.AssetID{assets[0]},
		false,
		map[bc.AssetID]int64{assets[0]: 5},
	}, {
		[]bc.AssetID{assets[0]},
		true,
		map[bc.AssetID]int64{assets[0]: 5},
	}, {
		[]bc.AssetID{assets[1]},
		false,
		map[bc.AssetID]int64{assets[1]: 7},
	}, {
		[]bc.AssetID{assets[1]},
		true,
		map[bc.AssetID]int64{assets[1]: 7},
	}, {
		[]bc.AssetID{assets[2]},
		false,
		map[bc.AssetID]int64{assets[2]: 8},
	}, {
		[]bc.AssetID{assets[2]},
		true,
		map[bc.AssetID]int64{assets[2]: -3},
	}, {
		[]bc.AssetID{assets[0], assets[1], assets[2]},
		false,
		map[bc.AssetID]int64{assets[0]: 5, assets[1]: 7, assets[2]: 8},
	}, {
		[]bc.AssetID{assets[0], assets[1], assets[2]},
		true,
		map[bc.AssetID]int64{assets[0]: 5, assets[1]: 7, assets[2]: -3},
	}, {
		[]bc.AssetID{assets[0], assets[3], assets[4]},
		false,
		map[bc.AssetID]int64{assets[0]: 5},
	}, {
		[]bc.AssetID{assets[0], assets[3], assets[3]},
		true,
		map[bc.AssetID]int64{assets[0]: 5, assets[3]: 4},
	}, {
		[]bc.AssetID{assets[4]},
		false,
		map[bc.AssetID]int64{},
	}, {
		[]bc.AssetID{assets[4]},
		true,
		map[bc.AssetID]int64{},
	}}

	for _, c := range cases {
		got, err := circulation(ctx, dbtx, c.isPool, c.aids)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("got Circulation(%+v) = %+v want %+v", c.aids, got, c.want)
		}
	}
}
