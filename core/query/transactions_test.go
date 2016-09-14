package query

import (
	"context"
	"math"
	"reflect"
	"testing"

	"chain/core/query/filter"
	"chain/database/pg/pgtest"
	"chain/protocol"
)

func TestLookupTxAfterNoBlocks(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)
	indexer := NewIndexer(db, &protocol.Chain{})

	cur, err := indexer.LookupTxAfter(ctx, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := TxAfter{
		MaxBlockHeight: 0,
		MaxPosition:    math.MaxInt32,
		MinBlockHeight: 0,
	}
	if !reflect.DeepEqual(cur, want) {
		t.Errorf("Got tx after %s, want %s", cur, want)
	}
}

func TestConstructTransactionsQuery(t *testing.T) {
	testCases := []struct {
		filter     string
		values     []interface{}
		after      TxAfter
		wantQuery  string
		wantValues []interface{}
	}{
		{
			filter:    `inputs(action='issue' AND asset_id=$1)`,
			values:    []interface{}{"abc"},
			after:     TxAfter{MaxBlockHeight: 205, MaxPosition: 35, MinBlockHeight: 100},
			wantQuery: `SELECT block_height, tx_pos, data FROM annotated_txs WHERE (data @> $1::jsonb) AND (block_height, tx_pos) <= ($2, $3) AND block_height >= $4 ORDER BY block_height DESC, tx_pos DESC LIMIT 100`,
			wantValues: []interface{}{
				`{"inputs":[{"action":"issue","asset_id":"abc"}]}`,
				uint64(205), uint32(35), uint64(100),
			},
		},
		{
			filter:    `outputs(account_id = $1 OR reference_data.corporate=$2)`,
			values:    []interface{}{"acc123", "corp"},
			after:     TxAfter{MaxBlockHeight: 2, MaxPosition: 20, MinBlockHeight: 1},
			wantQuery: `SELECT block_height, tx_pos, data FROM annotated_txs WHERE ((data @> $1::jsonb) OR (data @> $2::jsonb)) AND (block_height, tx_pos) <= ($3, $4) AND block_height >= $5 ORDER BY block_height DESC, tx_pos DESC LIMIT 100`,
			wantValues: []interface{}{
				`{"outputs":[{"account_id":"acc123"}]}`,
				`{"outputs":[{"reference_data":{"corporate":"corp"}}]}`,
				uint64(2), uint32(20), uint64(1),
			},
		},
	}

	for _, tc := range testCases {
		f, err := filter.Parse(tc.filter)
		if err != nil {
			t.Fatal(err)
		}
		expr, err := filter.AsSQL(f, "data", tc.values)
		if err != nil {
			t.Fatal(err)
		}

		query, values := constructTransactionsQuery(expr, tc.after, 100)
		if query != tc.wantQuery {
			t.Errorf("got\n%s\nwant\n%s", query, tc.wantQuery)
		}
		if !reflect.DeepEqual(values, tc.wantValues) {
			t.Errorf("got %#v, want %#v", values, tc.wantValues)
		}
	}
}
