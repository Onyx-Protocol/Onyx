package query

import (
	"reflect"
	"testing"

	"chain/core/query/chql"
)

func TestConstructBalancesQuery(t *testing.T) {
	testCases := []struct {
		query      string
		values     []interface{}
		wantQuery  string
		wantValues []interface{}
	}{
		{
			query:      "asset_id = $1 AND account_id = 'abc'",
			wantQuery:  `SELECT SUM((data->>'amount')::integer) AS balance, "data"->>'asset_id' FROM "annotated_outputs" WHERE (data @> $1::jsonb) GROUP BY 2`,
			wantValues: []interface{}{`{"account_id":"abc"}`},
		},
		{
			query:      "asset_id = $1 AND account_id = $2",
			values:     []interface{}{"foo", "bar"},
			wantQuery:  `SELECT SUM((data->>'amount')::integer) AS balance FROM "annotated_outputs" WHERE (data @> $1::jsonb)`,
			wantValues: []interface{}{`{"account_id":"bar","asset_id":"foo"}`},
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
		query, values, err := constructBalancesQuery(expr)
		if err != nil {
			t.Fatal(err)
		}

		if query != tc.wantQuery {
			t.Errorf("got %s want %s", query, tc.wantQuery)
		}
		if !reflect.DeepEqual(values, tc.wantValues) {
			t.Errorf("got %#v, want %#v", values, tc.wantValues)
		}
	}
}
