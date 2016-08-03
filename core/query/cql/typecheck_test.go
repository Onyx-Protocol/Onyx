package cql

import "testing"

var tbl = SQLTable{
	"is_issuance":     {"is_issuance", Bool},
	"asset_id":        {"asset_id", String},
	"account_id":      {"account_id", String},
	"amount":          {"amount", Integer},
	"account_tags":    {"account_tags", List},
	"account_numbers": {"account_numbers", List},
}

func TestTypeCheckInvalid(t *testing.T) {
	testCases := []struct {
		cql   string
		table SQLTable
	}{
		{cql: `1 = 'hello world'`},
		{cql: `NOT 1`},
		{cql: `'chain' >= 1`},
		{cql: `INPUTS('hello')`},
		{cql: `'hello' CONTAINS 'ello'`},
		{cql: `asset_id >= 5`, table: tbl},
		{cql: `is_issuance AND amount`, table: tbl},
		{cql: `account_tags > 'world'`, table: tbl},
		{cql: `account_tags CONTAINS account_tags`, table: tbl},
		{cql: `'hello' OR account_tags CONTAINS $1`, table: tbl},
	}

	for _, tc := range testCases {
		expr, _, err := parse(tc.cql)
		if err != nil {
			t.Fatal(err)
		}

		typ, err := typeCheckExpr(expr, tc.table)
		if err == nil {
			t.Errorf("typeCheckExpr(%s) = %s, want error", expr, typ)
		}
	}
}

func TestTypeCheckValid(t *testing.T) {
	testCases := []struct {
		cql   string
		typ   Type
		table SQLTable
	}{
		{cql: `1`, typ: Integer},
		{cql: `'hello world'`, typ: String},
		{cql: `1 = 1`, typ: Bool},
		{cql: `1 >= 1`, typ: Bool},
		{cql: `1 != 1`, typ: Bool},
		{cql: `$1 = '292 Ivy St'`, typ: Bool},
		{cql: `NOT is_issuance`, typ: Bool, table: tbl},
		{cql: `'hello' = 'world'`, typ: Bool},
		{cql: `'hello' > 'world'`, typ: Bool},
		{cql: `$1 = 'hello' OR account_tags CONTAINS $1`, typ: Bool},
		{cql: `($1 = 'hello') OR (account_tags CONTAINS $1)`, typ: Bool},
		{cql: `inputs(account_tags CONTAINS 'domestic' AND account_tags CONTAINS 'revolving')`, typ: Bool},
		{cql: `is_issuance`, typ: Bool, table: tbl},
		{cql: `asset_id`, typ: String, table: tbl},
		{cql: `account_tags`, typ: List, table: tbl},
		{cql: `amount >= 5`, typ: Bool, table: tbl},
	}

	for _, tc := range testCases {
		expr, _, err := parse(tc.cql)
		if err != nil {
			t.Fatal(err)
		}

		typ, err := typeCheckExpr(expr, tc.table)
		if err != nil {
			t.Fatal(err)
		}
		if typ != tc.typ {
			t.Errorf("typeCheckExpr(%s) = %s, want %s", expr, typ, tc.typ)
		}
	}
}
