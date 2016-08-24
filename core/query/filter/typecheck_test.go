package filter

import "testing"

func TestTypeCheckInvalid(t *testing.T) {
	testCases := []struct {
		p string
	}{
		{p: `1 = 'hello world'`},
		{p: `INPUTS('hello')`},
		{p: `foo(1=1).bar`},
		{p: `'hello'.foo`},
	}

	for _, tc := range testCases {
		expr, _, err := parse(tc.p)
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
		p   string
		typ Type
	}{
		{p: `1`, typ: Integer},
		{p: `'hello world'`, typ: String},
		{p: `1 = 1`, typ: Bool},
		{p: `$1 = '292 Ivy St'`, typ: Bool},
		{p: `'hello' = 'world'`, typ: Bool},
		{p: `$1 = 'hello' OR account_tags.something = $1`, typ: Bool},
		{p: `($1 = 'hello') OR (account_tags.something = $1)`, typ: Bool},
		{p: `inputs(account_tags.domestic AND account_tags.type = 'revolving')`, typ: Bool},
	}

	for _, tc := range testCases {
		expr, _, err := parse(tc.p)
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
