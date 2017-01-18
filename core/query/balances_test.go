package query

import (
	"reflect"
	"testing"

	"chain/core/query/filter"
)

func TestConstructBalancesQuery(t *testing.T) {
	now := uint64(123456)
	testCases := []struct {
		predicate  string
		sumBy      []string
		values     []interface{}
		wantQuery  string
		wantValues []interface{}
	}{
		{
			predicate:  "account_id = 'abc'",
			sumBy:      []string{"asset_id"},
			wantQuery:  `SELECT COALESCE(SUM((data->>'amount')::bigint), 0), "data"->>'asset_id' FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8 GROUP BY 2`,
			wantValues: []interface{}{`{"account_id":"abc"}`, now},
		},
		{
			predicate:  "account_id = $1",
			sumBy:      []string{"asset_id"},
			values:     []interface{}{"abc"},
			wantQuery:  `SELECT COALESCE(SUM((data->>'amount')::bigint), 0), "data"->>'asset_id' FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8 GROUP BY 2`,
			wantValues: []interface{}{`{"account_id":"abc"}`, now},
		},
		{
			predicate:  "asset_id = $1 AND account_id = $2",
			values:     []interface{}{"foo", "bar"},
			wantQuery:  `SELECT COALESCE(SUM((data->>'amount')::bigint), 0) FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8`,
			wantValues: []interface{}{`{"account_id":"bar","asset_id":"foo"}`, now},
		},
		{
			predicate:  "account_id = $1",
			sumBy:      []string{"asset_tags.currency"},
			values:     []interface{}{"foo"},
			wantQuery:  `SELECT COALESCE(SUM((data->>'amount')::bigint), 0), "data"->'asset_tags'->>'currency' FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8 GROUP BY 2`,
			wantValues: []interface{}{`{"account_id":"foo"}`, now},
		},
	}

	for i, tc := range testCases {
		p, err := filter.Parse(tc.predicate)
		if err != nil {
			t.Fatal(err)
		}
		expr, err := filter.AsSQL(p, "data", tc.values)
		if err != nil {
			t.Fatal(err)
		}
		var fields []filter.Field
		for _, s := range tc.sumBy {
			f, err := filter.ParseField(s)
			if err != nil {
				t.Fatal(err)
			}
			fields = append(fields, f)
		}

		query, values := constructBalancesQuery(expr, fields, now)
		if query != tc.wantQuery {
			t.Errorf("case %d: got\n%s\nwant\n%s", i, query, tc.wantQuery)
		}
		if !reflect.DeepEqual(values, tc.wantValues) {
			t.Errorf("case %d: got %#v, want %#v", i, values, tc.wantValues)
		}
	}
}
