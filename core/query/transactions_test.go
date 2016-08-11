package query

import (
	"reflect"
	"testing"

	"chain/core/query/chql"
)

func TestConstructTransactionsQuery(t *testing.T) {
	testCases := []struct {
		query      string
		values     []interface{}
		cursor     TxCursor
		wantQuery  string
		wantValues []interface{}
	}{
		{
			query:     `inputs(action='issue' AND asset_id=$1)`,
			values:    []interface{}{"abc"},
			cursor:    TxCursor{MaxBlockHeight: 205, MaxPosition: 35, MinBlockHeight: 100},
			wantQuery: `SELECT block_height, tx_pos, data FROM annotated_txs WHERE ((data @> $1::jsonb)) AND (block_height, tx_pos) <= ($2, $3) AND block_height >= $4 ORDER BY block_height DESC, tx_pos DESC LIMIT 100`,
			wantValues: []interface{}{
				`{"inputs":[{"action":"issue","asset_id":"abc"}]}`,
				uint64(205), uint32(35), uint64(100),
			},
		},
		{
			query:     `outputs(account_id = $1 OR reference_data.corporate=$2)`,
			values:    []interface{}{"acc123", "corp"},
			cursor:    TxCursor{MaxBlockHeight: 2, MaxPosition: 20, MinBlockHeight: 1},
			wantQuery: `SELECT block_height, tx_pos, data FROM annotated_txs WHERE ((data @> $1::jsonb) OR (data @> $2::jsonb)) AND (block_height, tx_pos) <= ($3, $4) AND block_height >= $5 ORDER BY block_height DESC, tx_pos DESC LIMIT 100`,
			wantValues: []interface{}{
				`{"outputs":[{"account_id":"acc123"}]}`,
				`{"outputs":[{"reference_data":{"corporate":"corp"}}]}`,
				uint64(2), uint32(20), uint64(1),
			},
		},
	}

	for _, tc := range testCases {
		q, err := chql.Parse(tc.query)
		if err != nil {
			t.Fatal(err)
		}
		expr, err := chql.AsSQL(q, "data", tc.values)
		if err != nil {
			t.Fatal(err)
		}

		query, values := constructTransactionsQuery(expr, tc.cursor, 100)
		if query != tc.wantQuery {
			t.Errorf("got\n%s\nwant\n%s", query, tc.wantQuery)
		}
		if !reflect.DeepEqual(values, tc.wantValues) {
			t.Errorf("got %#v, want %#v", values, tc.wantValues)
		}
	}
}
