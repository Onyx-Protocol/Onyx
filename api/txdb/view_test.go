package txdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg/pgtest"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

func TestView(t *testing.T) {
	const fix = `
		INSERT INTO utxos
			(tx_hash, index, asset_id, amount, script, metadata)
		VALUES
			('1000000000000000000000000000000000000000000000000000000000000000', 1, 'A55E710000000000000000000000000000000000000000000000000000000000', 1, 'script-1', 'metadata-1'),
			('2000000000000000000000000000000000000000000000000000000000000000', 2, 'A55E720000000000000000000000000000000000000000000000000000000000', 2, 'script-2', 'metadata-2');

		INSERT INTO blocks_utxos (tx_hash, index)
		VALUES
			('1000000000000000000000000000000000000000000000000000000000000000', 1),
			('2000000000000000000000000000000000000000000000000000000000000000', 2);
	`

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

	withContext(t, fix, func(ctx context.Context) {
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
	})
}

func TestViewForPrevoutsIgnoreIssuance(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

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
	const fix = `
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
	`

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

	withContext(t, fix, func(ctx context.Context) {
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
	})
}
