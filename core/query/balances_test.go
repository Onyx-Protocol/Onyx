package query

import (
	"testing"

	"chain/core/query/filter"
	"chain/testutil"
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
			wantQuery:  `SELECT COALESCE(SUM(amount), 0), encode(out."asset_id", 'hex') FROM "annotated_outputs" AS out WHERE (out."account_id" = 'abc') AND timespan @> $1::int8 GROUP BY 2`,
			wantValues: []interface{}{now},
		},
		{
			predicate:  "account_id = $1",
			sumBy:      []string{"asset_id"},
			values:     []interface{}{"abc"},
			wantQuery:  `SELECT COALESCE(SUM(amount), 0), encode(out."asset_id", 'hex') FROM "annotated_outputs" AS out WHERE (out."account_id" = $1) AND timespan @> $2::int8 GROUP BY 2`,
			wantValues: []interface{}{`abc`, now},
		},
		{
			predicate:  "asset_id = $1 AND account_id = $2",
			values:     []interface{}{"foo", "bar"},
			wantQuery:  `SELECT COALESCE(SUM(amount), 0) FROM "annotated_outputs" AS out WHERE (encode(out."asset_id", 'hex') = $1 AND out."account_id" = $2) AND timespan @> $3::int8`,
			wantValues: []interface{}{`foo`, `bar`, now},
		},
		{
			predicate:  "account_id = $1",
			sumBy:      []string{"asset_tags.currency"},
			values:     []interface{}{"foo"},
			wantQuery:  `SELECT COALESCE(SUM(amount), 0), out."asset_tags"->>'currency' FROM "annotated_outputs" AS out WHERE (out."account_id" = $1) AND timespan @> $2::int8 GROUP BY 2`,
			wantValues: []interface{}{`foo`, now},
		},
	}

	for i, tc := range testCases {
		p, err := filter.Parse(tc.predicate, outputsTable, tc.values)
		if err != nil {
			t.Fatal(err)
		}
		expr, err := filter.AsSQL(p, outputsTable, tc.values)
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

		query, values, err := constructBalancesQuery(expr, tc.values, fields, now)
		if err != nil {
			t.Fatal(err)
		}
		if query != tc.wantQuery {
			t.Errorf("case %d: got\n%s\nwant\n%s", i, query, tc.wantQuery)
		}
		if !testutil.DeepEqual(values, tc.wantValues) {
			t.Errorf("case %d: got %#v, want %#v", i, values, tc.wantValues)
		}
	}
}
