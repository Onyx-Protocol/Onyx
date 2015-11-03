package txdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/fedchain/state"
)

func TestView(t *testing.T) {
	const fix = `
		INSERT INTO utxos
			(txid, index, asset_id, amount, addr_index, account_id, manager_node_id, script, metadata)
		VALUES
			('1000000000000000000000000000000000000000000000000000000000000000', 1, 'A55E710000000000000000000000000000000000000000000000000000000000', 1, 0, 'account-1', 'mnode-1', 'script-1', 'metadata-1'),
			('2000000000000000000000000000000000000000000000000000000000000000', 2, 'A55E720000000000000000000000000000000000000000000000000000000000', 2, 0, 'account-2', 'mnode-2', 'script-2', 'metadata-2');
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
					AssetID:  bc.AssetID(mustParseHash("A55E710000000000000000000000000000000000000000000000000000000000")),
					Value:    1,
					Script:   []byte("script-1"),
					Metadata: []byte("metadata-1"),
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
					AssetID:  bc.AssetID(mustParseHash("A55E720000000000000000000000000000000000000000000000000000000000")),
					Value:    2,
					Script:   []byte("script-2"),
					Metadata: []byte("metadata-2"),
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

	withContext(t, fix, func(t *testing.T, ctx context.Context) {
		for i, ex := range examples {
			t.Log("Example", i)

			var verr error
			v := NewView(&verr)

			got := v.Output(ctx, ex.op)

			if verr != nil {
				t.Fatal("unexpected error:", verr)
			}

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("output:\ngot:  %v\nwant: %v", got, ex.want)
			}
		}
	})
}

func TestPoolView(t *testing.T) {
	const fix = `
		INSERT INTO pool_txs
			(tx_hash, data)
		VALUES
			('1000000000000000000000000000000000000000000000000000000000000000', ''),
			('2000000000000000000000000000000000000000000000000000000000000000', ''),
			('3000000000000000000000000000000000000000000000000000000000000000', '');

		INSERT INTO pool_outputs
			(tx_hash, index, asset_id, amount, addr_index, account_id, manager_node_id, script, metadata)
		VALUES
			('1000000000000000000000000000000000000000000000000000000000000000', 1, 'A55E710000000000000000000000000000000000000000000000000000000000', 1, 0, 'account-1', 'mnode-1', 'script-1', 'metadata-1'),
			('2000000000000000000000000000000000000000000000000000000000000000', 2, 'A55E720000000000000000000000000000000000000000000000000000000000', 2, 0, 'account-2', 'mnode-2', 'script-2', 'metadata-2'),
			('3000000000000000000000000000000000000000000000000000000000000000', 3, 'A55E730000000000000000000000000000000000000000000000000000000000', 3, 0, 'account-3', 'mnode-3', 'script-3', 'metadata-3');

		INSERT INTO pool_inputs
			(tx_hash, index)
		VALUES
			('3000000000000000000000000000000000000000000000000000000000000000', 3);
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
					AssetID:  bc.AssetID(mustParseHash("A55E710000000000000000000000000000000000000000000000000000000000")),
					Value:    1,
					Script:   []byte("script-1"),
					Metadata: []byte("metadata-1"),
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
					AssetID:  bc.AssetID(mustParseHash("A55E720000000000000000000000000000000000000000000000000000000000")),
					Value:    2,
					Script:   []byte("script-2"),
					Metadata: []byte("metadata-2"),
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
				TxOutput: bc.TxOutput{},
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
			want: nil,
		},
	}

	withContext(t, fix, func(t *testing.T, ctx context.Context) {
		for i, ex := range examples {
			t.Log("Example", i)

			var verr error
			v := NewPoolView(&verr)

			got := v.Output(ctx, ex.op)

			if verr != nil {
				t.Fatal("unexpected error:", verr)
			}

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("output:\ngot:  %v\nwant: %v", got, ex.want)
			}
		}
	})
}
