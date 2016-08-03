package cql

import (
	"reflect"
	"testing"
)

func TestTranslateToSQL(t *testing.T) {
	cols := SQLTable{
		"asset_id":        {"asset_id", String},
		"amount":          {"amount", Integer},
		"account_id":      {"account_id", String},
		"account_tags":    {"account_tags", List},
		"account_numbers": {"account_numbers", List},
	}

	testCases := []struct {
		cql  string
		sql  string
		vals []sqlPlaceholder
	}{
		{cql: `2000`, sql: `2000`},
		{cql: `0xFF`, sql: `255`},
		{cql: `'usd'`, sql: `$1`, vals: []sqlPlaceholder{{value: "usd"}}},
		{cql: `1 OR 1`, sql: `(1 OR 1)`},
		{cql: `1 AND 1`, sql: `(1 AND 1)`},
		{cql: `4 < 5`, sql: `(4 < 5)`},
		{cql: `4 <= 5`, sql: `(4 <= 5)`},
		{cql: `4 > 5`, sql: `(4 > 5)`},
		{cql: `4 >= 5`, sql: `(4 >= 5)`},
		{cql: `4 = 5`, sql: `(4 = 5)`},
		{cql: `4 != 5`, sql: `(4 != 5)`},
		{cql: `asset_id`, sql: `"asset_id"`},
		{cql: `account_numbers CONTAINS 20525`, sql: `("account_numbers" @> ARRAY[20525])`},
		{
			cql:  `account_tags CONTAINS 'high-roller'`,
			sql:  `("account_tags" @> ARRAY[$1])`,
			vals: []sqlPlaceholder{{value: "high-roller"}},
		},
		{
			cql:  `account_id = 'abc' OR account_tags CONTAINS $1`,
			sql:  `(("account_id" = $1) OR ("account_tags" @> ARRAY[$2]))`,
			vals: []sqlPlaceholder{{value: "abc"}, {number: 1}},
		},
		{
			cql:  `(account_id = $1) AND (amount > 2000) AND (asset_id = $2)`,
			sql:  `((("account_id" = $1) AND ("amount" > 2000)) AND ("asset_id" = $2))`,
			vals: []sqlPlaceholder{{number: 1}, {number: 2}},
		},
	}

	for _, tc := range testCases {
		var sqlExpr SQLExpr

		expr, _, err := parse(tc.cql)
		if err != nil {
			t.Fatal(err)
		}

		translateToSQL(&sqlExpr, cols, expr)
		got := sqlExpr.String()
		if got != tc.sql {
			t.Errorf("translateToSQL(%q) = %q, want %q", tc.cql, got, tc.sql)
		}
		if !reflect.DeepEqual(sqlExpr.placeholders, tc.vals) {
			t.Errorf("translateToSQL(%q) values %#v, want %#v", tc.cql, sqlExpr.placeholders, tc.vals)
		}
	}
}
