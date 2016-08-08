package chql

import "testing"

var tbl = SQLTable{
	"is_issuance":     {"is_issuance", Bool},
	"asset_id":        {"asset_id", String},
	"account_id":      {"account_id", String},
	"amount":          {"amount", Integer},
	"account_tags":    {"account_tags", Object},
	"account_numbers": {"account_numbers", Object},
	"reference":       {"reference_data", Object},
}

func TestTypeCheckInvalid(t *testing.T) {
	testCases := []struct {
		chql  string
		table SQLTable
	}{
		{chql: `1 = 'hello world'`},
		{chql: `NOT 1`},
		{chql: `'chain' >= 1`},
		{chql: `INPUTS('hello')`},
		{chql: `foo(1=1).bar`},
		{chql: `'hello'.foo`},
		{chql: `asset_id >= 5`, table: tbl},
		{chql: `is_issuance AND amount`, table: tbl},
		{chql: `account_tags > 'world'`, table: tbl},
		{chql: `account_tags = account_tags`, table: tbl},
		{chql: `'hello' OR account_tags.foo = $1`, table: tbl},
	}

	for _, tc := range testCases {
		expr, _, err := parse(tc.chql)
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
		chql  string
		typ   Type
		table SQLTable
	}{
		{chql: `1`, typ: Integer},
		{chql: `'hello world'`, typ: String},
		{chql: `1 = 1`, typ: Bool},
		{chql: `1 >= 1`, typ: Bool},
		{chql: `1 != 1`, typ: Bool},
		{chql: `$1 = '292 Ivy St'`, typ: Bool},
		{chql: `NOT is_issuance`, typ: Bool, table: tbl},
		{chql: `'hello' = 'world'`, typ: Bool},
		{chql: `'hello' > 'world'`, typ: Bool},
		{chql: `$1 = 'hello' OR account_tags.something = $1`, typ: Bool},
		{chql: `($1 = 'hello') OR (account_tags.something = $1)`, typ: Bool},
		{chql: `inputs(account_tags.domestic AND account_tags.type = 'revolving')`, typ: Bool},
		{chql: `is_issuance`, typ: Bool, table: tbl},
		{chql: `asset_id`, typ: String, table: tbl},
		{chql: `account_tags`, typ: Object, table: tbl},
		{chql: `amount >= 5`, typ: Bool, table: tbl},
		{chql: `reference.recipient.id`, typ: Any, table: tbl},
	}

	for _, tc := range testCases {
		expr, _, err := parse(tc.chql)
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
