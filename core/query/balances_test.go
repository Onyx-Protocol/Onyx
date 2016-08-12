package query

import (
	"reflect"
	"testing"

	"chain/core/query/chql"
)

func TestConstructBalancesQuery(t *testing.T) {
	now := uint64(123456)
	testCases := []struct {
		query      string
		values     []interface{}
		wantQuery  string
		wantValues []interface{}
	}{
		{
			query:      "asset_id = $1 AND account_id = 'abc'",
			wantQuery:  `SELECT COALESCE(SUM((data->>'amount')::integer), 0), "data"->>'asset_id' FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8 GROUP BY 2`,
			wantValues: []interface{}{`{"account_id":"abc"}`, now},
		},
		{
			query:      "asset_id = $1 AND account_id = $2",
			values:     []interface{}{"foo", "bar"},
			wantQuery:  `SELECT COALESCE(SUM((data->>'amount')::integer), 0) FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8`,
			wantValues: []interface{}{`{"account_id":"bar","asset_id":"foo"}`, now},
		},
		{
			query:      "account_id = $1 AND asset_tags.currency = $2",
			values:     []interface{}{"foo"},
			wantQuery:  `SELECT COALESCE(SUM((data->>'amount')::integer), 0), "data"->'asset_tags'->'currency' FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8 GROUP BY 2`,
			wantValues: []interface{}{`{"account_id":"foo"}`, now},
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
		query, values := constructBalancesQuery(expr, now)
		if query != tc.wantQuery {
			t.Errorf("case %d: got\n%s\nwant\n%s", i, query, tc.wantQuery)
		}
		if !reflect.DeepEqual(values, tc.wantValues) {
			t.Errorf("case %d: got %#v, want %#v", i, values, tc.wantValues)
		}
	}
}
