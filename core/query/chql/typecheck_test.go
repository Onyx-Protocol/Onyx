package chql

import "testing"

func TestTypeCheckInvalid(t *testing.T) {
	testCases := []struct {
		chql string
	}{
		{chql: `1 = 'hello world'`},
		{chql: `INPUTS('hello')`},
		{chql: `foo(1=1).bar`},
		{chql: `'hello'.foo`},
	}

	for _, tc := range testCases {
		expr, _, err := parse(tc.chql)
		if err != nil {
			t.Fatal(err)
		}

		typ, err := typeCheckExpr(expr)
		if err == nil {
			t.Errorf("typeCheckExpr(%s) = %s, want error", expr, typ)
		}
	}
}

func TestTypeCheckValid(t *testing.T) {
	testCases := []struct {
		chql string
		typ  Type
	}{
		{chql: `1`, typ: Integer},
		{chql: `'hello world'`, typ: String},
		{chql: `1 = 1`, typ: Bool},
		{chql: `$1 = '292 Ivy St'`, typ: Bool},
		{chql: `'hello' = 'world'`, typ: Bool},
		{chql: `$1 = 'hello' OR account_tags.something = $1`, typ: Bool},
		{chql: `($1 = 'hello') OR (account_tags.something = $1)`, typ: Bool},
		{chql: `inputs(account_tags.domestic AND account_tags.type = 'revolving')`, typ: Bool},
	}

	for _, tc := range testCases {
		expr, _, err := parse(tc.chql)
		if err != nil {
			t.Fatal(err)
		}

		typ, err := typeCheckExpr(expr)
		if err != nil {
			t.Fatal(err)
		}
		if typ != tc.typ {
			t.Errorf("typeCheckExpr(%s) = %s, want %s", expr, typ, tc.typ)
		}
	}
}
