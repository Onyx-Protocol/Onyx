package chql

import (
	"reflect"
	"testing"
)

func TestTranslateToSQL(t *testing.T) {
	cols := SQLTable{
		"asset_id":        {"asset_id", String},
		"amount":          {"amount", Integer},
		"account_id":      {"account_id", String},
		"account_tags":    {"account_tags", Object},
		"account_numbers": {"account_numbers", Object},
	}

	testCases := []struct {
		chql string
		sql  string
		vals []sqlPlaceholder
	}{
		{chql: `2000`, sql: `2000`},
		{chql: `0xFF`, sql: `255`},
		{chql: `'usd'`, sql: `$1`, vals: []sqlPlaceholder{{value: "usd"}}},
		{chql: `1 OR 1`, sql: `(1 OR 1)`},
		{chql: `1 AND 1`, sql: `(1 AND 1)`},
		{chql: `4 < 5`, sql: `(4 < 5)`},
		{chql: `4 <= 5`, sql: `(4 <= 5)`},
		{chql: `4 > 5`, sql: `(4 > 5)`},
		{chql: `4 >= 5`, sql: `(4 >= 5)`},
		{chql: `4 = 5`, sql: `(4 = 5)`},
		{chql: `4 != 5`, sql: `(4 != 5)`},
		{chql: `asset_id`, sql: `"asset_id"`},
		{
			chql: `(account_id = $1) AND (amount > 2000) AND (asset_id = $2)`,
			sql:  `((("account_id" = $1) AND ("amount" > 2000)) AND ("asset_id" = $2))`,
			vals: []sqlPlaceholder{{number: 1}, {number: 2}},
		},
	}

	for _, tc := range testCases {
		var sqlExpr SQLExpr

		expr, _, err := parse(tc.chql)
		if err != nil {
			t.Fatal(err)
		}

		translateToSQL(&sqlExpr, cols, expr)
		got := sqlExpr.String()
		if got != tc.sql {
			t.Errorf("translateToSQL(%q) = %q, want %q", tc.chql, got, tc.sql)
		}
		if !reflect.DeepEqual(sqlExpr.placeholders, tc.vals) {
			t.Errorf("translateToSQL(%q) values %#v, want %#v", tc.chql, sqlExpr.placeholders, tc.vals)
		}
	}
}
