package txdb

import (
	"reflect"
	"testing"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestView(t *testing.T) {
	ctx := pgtest.NewContext(t)
	pgtest.Exec(ctx, t, `
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
		op   bc.Outpoint
		want *state.Output
	}{
		{
			op: bc.Outpoint{
				Hash:  mustParseHash("1000000000000000000000000000000000000000000000000000000000000000"),
				Index: 1,
			},
			want: &state.Output{
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
			op: bc.Outpoint{
				Hash:  mustParseHash("2000000000000000000000000000000000000000000000000000000000000000"),
				Index: 2,
			},
			want: &state.Output{
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
			op: bc.Outpoint{
				Hash:  mustParseHash("3000000000000000000000000000000000000000000000000000000000000000"),
				Index: 3,
			},
			want: nil,
		},
	}
	for i, ex := range examples {
		t.Log("Example", i)

		v, err := newView(ctx, []bc.Outpoint{ex.op})
		if err != nil {
			t.Fatal("unexpected error:", err)
		}

		got := v.Output(ctx, ex.op)

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("output:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestViewForPrevoutsIgnoreIssuance(t *testing.T) {
	ctx := pgtest.NewContext(t)

	txs := []*bc.Tx{bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{{
			Previous: bc.Outpoint{
				Index: 0xffffffff,
			},
		}},
	})}

	v, err := newViewForPrevouts(ctx, txs)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	got := len(v.(*view).outs)
	if got != 0 {
		t.Errorf("len(outs) = %d want 0", got)
	}
}

func TestPoolView(t *testing.T) {
	ctx := pgtest.NewContext(t)
	pgtest.Exec(ctx, t, `
		INSERT INTO pool_txs
			(tx_hash, data)
		VALUES
			('1000000000000000000000000000000000000000000000000000000000000000', ''),
			('2000000000000000000000000000000000000000000000000000000000000000', ''),
			('3000000000000000000000000000000000000000000000000000000000000000', '');

		INSERT INTO utxos
			(tx_hash, index, asset_id, amount, script, metadata)
		VALUES
			('1000000000000000000000000000000000000000000000000000000000000000', 1, 'A55E710000000000000000000000000000000000000000000000000000000000', 1, 'script-1', 'metadata-1'),
			('2000000000000000000000000000000000000000000000000000000000000000', 2, 'A55E720000000000000000000000000000000000000000000000000000000000', 2, 'script-2', 'metadata-2'),
			('3000000000000000000000000000000000000000000000000000000000000000', 3, 'A55E730000000000000000000000000000000000000000000000000000000000', 3, 'script-3', 'metadata-3');

		INSERT INTO pool_inputs
			(tx_hash, index)
		VALUES
			('3000000000000000000000000000000000000000000000000000000000000000', 3),
			('4000000000000000000000000000000000000000000000000000000000000000', 4);
	`)

	examples := []struct {
		op   bc.Outpoint
		want *state.Output
	}{
		{
			op: bc.Outpoint{
				Hash:  mustParseHash("1000000000000000000000000000000000000000000000000000000000000000"),
				Index: 1,
			},
			want: &state.Output{
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
			op: bc.Outpoint{
				Hash:  mustParseHash("2000000000000000000000000000000000000000000000000000000000000000"),
				Index: 2,
			},
			want: &state.Output{
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
			op: bc.Outpoint{
				Hash:  mustParseHash("3000000000000000000000000000000000000000000000000000000000000000"),
				Index: 3,
			},
			want: &state.Output{
				TxOutput: bc.TxOutput{
					AssetAmount: bc.AssetAmount{AssetID: bc.AssetID(mustParseHash("A55E730000000000000000000000000000000000000000000000000000000000")), Amount: 3},
					Script:      []byte("script-3"),
					Metadata:    []byte("metadata-3"),
				},
				Outpoint: bc.Outpoint{
					Hash:  mustParseHash("3000000000000000000000000000000000000000000000000000000000000000"),
					Index: 3,
				},
				Spent: true,
			},
		},
		{
			op: bc.Outpoint{
				Hash:  mustParseHash("4000000000000000000000000000000000000000000000000000000000000000"),
				Index: 4,
			},
			want: &state.Output{
				Outpoint: bc.Outpoint{
					Hash:  mustParseHash("4000000000000000000000000000000000000000000000000000000000000000"),
					Index: 4,
				},
				Spent: true,
			},
		},
		{
			op: bc.Outpoint{
				Hash:  mustParseHash("5000000000000000000000000000000000000000000000000000000000000000"),
				Index: 5,
			},
			want: nil,
		},
	}

	for i, ex := range examples {
		t.Log("Example", i)

		v, err := newPoolView(ctx, []bc.Outpoint{ex.op})
		if err != nil {
			t.Fatal("unexpected error:", err)
		}

		got := v.Output(ctx, ex.op)
		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("output:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestViewCirculation(t *testing.T) {
	ctx := pgtest.NewContext(t)

	assets := []bc.AssetID{{1}, {2}, {3}, {4}, {5}}
	err := addIssuances(ctx, map[bc.AssetID]*state.AssetState{
		assets[0]: &state.AssetState{Issuance: 5},
		assets[1]: &state.AssetState{Issuance: 9, Destroyed: 2},
		assets[2]: &state.AssetState{Issuance: 8},
	}, true)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = addIssuances(ctx, map[bc.AssetID]*state.AssetState{
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
		v := &view{isPool: c.isPool}
		got, err := v.Circulation(ctx, c.aids)
		if err != nil {
			testutil.FatalErr(t, err)
		}

		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("got Circulation(%+v) = %+v want %+v", c.aids, got, c.want)
		}
	}
}
