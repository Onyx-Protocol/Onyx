package query

import (
	"context"
	"math"
	"reflect"
	"testing"

	"chain/core/query/filter"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/protocol"
)

func TestDecodeTxAfter(t *testing.T) {
	testCases := []struct {
		str     string
		want    TxAfter
		wantErr error
	}{
		{
			"1:0-2",
			TxAfter{
				FromBlockHeight: 1,
				FromPosition:    0,
				StopBlockHeight: 2,
			},
			nil,
		},
		{
			"hello",
			TxAfter{},
			ErrBadAfter,
		},
	}

	for _, c := range testCases {
		got, err := DecodeTxAfter(c.str)
		if errors.Root(err) != c.wantErr {
			t.Fatalf("DecodeTxAfter(%q) unexpected error %s, want %v", c.str, err, c.wantErr)
		}

		if got != c.want {
			t.Fatalf("want DecodeTxAfter(%q)=%#v, got %#v", c.str, c.want, got)
		}
	}
}

func TestLookupTxAfterNoBlocks(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)
	indexer := NewIndexer(db, &protocol.Chain{}, nil)

	cur, err := indexer.LookupTxAfter(ctx, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := TxAfter{
		FromBlockHeight: 0,
		FromPosition:    math.MaxInt32,
		StopBlockHeight: 0,
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
		asc        bool
		wantQuery  string
		wantValues []interface{}
	}{
		{
			filter:    `inputs(type='issue' AND asset_id=$1)`,
			values:    []interface{}{"abc"},
			after:     TxAfter{FromBlockHeight: 205, FromPosition: 35, StopBlockHeight: 100},
			asc:       false,
			wantQuery: `SELECT block_height, tx_pos, data FROM annotated_txs WHERE (data @> $1::jsonb) AND (block_height, tx_pos) < ($2, $3) AND block_height >= $4 ORDER BY block_height DESC, tx_pos DESC LIMIT 100`,
			wantValues: []interface{}{
				`{"inputs":[{"asset_id":"abc","type":"issue"}]}`,
				uint64(205), uint32(35), uint64(100),
			},
		},
		{
			filter:    `outputs(account_id = $1 OR reference_data.corporate=$2)`,
			values:    []interface{}{"acc123", "corp"},
			after:     TxAfter{FromBlockHeight: 2, FromPosition: 20, StopBlockHeight: 1},
			asc:       false,
			wantQuery: `SELECT block_height, tx_pos, data FROM annotated_txs WHERE ((data @> $1::jsonb) OR (data @> $2::jsonb)) AND (block_height, tx_pos) < ($3, $4) AND block_height >= $5 ORDER BY block_height DESC, tx_pos DESC LIMIT 100`,
			wantValues: []interface{}{
				`{"outputs":[{"account_id":"acc123"}]}`,
				`{"outputs":[{"reference_data":{"corporate":"corp"}}]}`,
				uint64(2), uint32(20), uint64(1),
			},
		},
		{
			filter:    `outputs(account_id = $1 OR reference_data.corporate=$2)`,
			values:    []interface{}{"acc123", "corp"},
			after:     TxAfter{FromBlockHeight: 2, FromPosition: 20, StopBlockHeight: 1},
			asc:       true,
			wantQuery: `SELECT block_height, tx_pos, data FROM annotated_txs WHERE ((data @> $1::jsonb) OR (data @> $2::jsonb)) AND (block_height, tx_pos) > ($3, $4) AND block_height <= $5 ORDER BY block_height ASC, tx_pos ASC LIMIT 100`,
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

		query, values := constructTransactionsQuery(expr, tc.after, tc.asc, 100)
		if query != tc.wantQuery {
			t.Errorf("got\n%s\nwant\n%s", query, tc.wantQuery)
		}
		if !reflect.DeepEqual(values, tc.wantValues) {
			t.Errorf("got %#v, want %#v", values, tc.wantValues)
		}
	}
}
