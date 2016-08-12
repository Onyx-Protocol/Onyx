package query

import (
	"reflect"
	"testing"
	"time"

	"chain/core/query/chql"
	"chain/cos/bc"
)

func TestConstructOutputsQuery(t *testing.T) {
	now := time.Unix(233400000, 0)
	nowMillis := bc.Millis(now)

	testCases := []struct {
		query      string
		values     []interface{}
		cursor     *OutputsCursor
		wantQuery  string
		wantValues []interface{}
	}{
		{
			query:      "asset_id = $1 AND account_id = 'abc'",
			values:     []interface{}{"foo"},
			wantQuery:  `SELECT block_height, tx_pos, output_index, data FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8 ORDER BY block_height ASC, tx_pos ASC, output_index ASC LIMIT 10`,
			wantValues: []interface{}{`{"account_id":"abc","asset_id":"foo"}`, nowMillis},
		},
		{
			query:  "asset_id = $1 AND account_id = 'abc'",
			values: []interface{}{"foo"},
			cursor: &OutputsCursor{
				lastBlockHeight: 15,
				lastTxPos:       17,
				lastIndex:       19,
			},
			wantQuery:  `SELECT block_height, tx_pos, output_index, data FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8 AND (block_height, tx_pos, output_index) > ($3, $4, $5) ORDER BY block_height ASC, tx_pos ASC, output_index ASC LIMIT 10`,
			wantValues: []interface{}{`{"account_id":"abc","asset_id":"foo"}`, nowMillis, uint64(15), uint32(17), uint32(19)},
		},
	}

	for i, tc := range testCases {
		q, err := chql.Parse(tc.query)
		if err != nil {
			t.Fatal(err)
		}
		expr, err := chql.AsSQL(q, "data", tc.values)
		if err != nil {
			t.Fatal(err)
		}
		query, values := constructOutputsQuery(expr, nowMillis, tc.cursor, 10)
		if query != tc.wantQuery {
			t.Errorf("case %d: got %s want %s", i, query, tc.wantQuery)
		}
		if !reflect.DeepEqual(values, tc.wantValues) {
			t.Errorf("case %d: got %#v, want %#v", i, values, tc.wantValues)
		}
	}
}
