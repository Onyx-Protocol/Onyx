package filter

import (
	"chain/testutil"
	"testing"
)

func TestTypeCheckInvalid(t *testing.T) {
	testCases := []struct {
		p string
	}{
		{p: `1 = 'hello world'`},
		{p: `inputs('hello')`},
		{p: `inputs(1=1).bar`},
		{p: `'hello'.foo`},
		{p: `inputs(amount = asset_id)`},
		{p: `inputs(100 = asset_id)`},
		{p: `inputs(wat)`},
		{p: `wat(asset_id = 'a')`},
		{p: `position(asset_id = 'a')`},
		{p: `position.huh`},
		{p: `ref.something = 'abc' OR ref.something = 123`},
		{p: `ref.buyer.id = 'abc' OR ref.buyer = 'hello'`},
	}

	for _, tc := range testCases {
		expr, _, err := parse(tc.p)
		if err != nil {
			t.Fatal(err)
		}

		m := make(map[string]Type)
		typ, err := typeCheckExpr(expr, transactionsSQLTable, nil, m)
		if err == nil {
			t.Errorf("typeCheckExpr(%s) = %s, want error", expr, typ)
		}
	}
}

func TestTypeCheckValid(t *testing.T) {
	testCases := []struct {
		p        string
		typ      Type
		valTypes []Type
	}{
		{p: `1`, typ: Integer},
		{p: `'hello world'`, typ: String},
		{p: `is_local`, typ: Bool},
		{p: `1 = 1`, typ: Bool},
		{p: `$1 = '292 Ivy St'`, typ: Bool, valTypes: []Type{String}},
		{p: `'hello' = 'world'`, typ: Bool},
		{p: `id = id`, typ: Bool},
		{p: `$1 = 'hello' OR ref.something = $1`, typ: Bool, valTypes: []Type{String}},
		{p: `($1 = 'hello') OR (ref.something = $1)`, typ: Bool, valTypes: []Type{String}},
		{p: `inputs(account_tags.domestic AND account_tags.type = 'revolving')`, typ: Bool},
		{p: `inputs(account_tags.state = account_tags.shipping_address.state)`, typ: Bool},
		{p: `$1`, valTypes: []Type{String}, typ: String},
		{p: `$1 = $2`, valTypes: []Type{String, String}, typ: Bool},
	}

	for _, tc := range testCases {
		expr, _, err := parse(tc.p)
		if err != nil {
			t.Fatal(err)
		}

		m := make(map[string]Type)
		typ, err := typeCheckExpr(expr, transactionsSQLTable, tc.valTypes, m)
		if err != nil {
			t.Fatal(err)
		}
		if typ != tc.typ {
			t.Errorf("typeCheckExpr(%s) = %s, want %s", expr, typ, tc.typ)
		}
	}
}

func TestTypeCheckSelector(t *testing.T) {
	const predicate = `ref.buyer.address.state = 'OH' AND inputs(account_tags.user_profile.id = 123)`

	expr, _, err := parse(predicate)
	if err != nil {
		t.Fatal(err)
	}

	m := make(map[string]Type)
	typ, err := typeCheckExpr(expr, transactionsSQLTable, nil, m)
	if err != nil {
		t.Fatal(err)
	}
	if typ != Bool {
		t.Errorf("typeCheckExpr(%s) = %s, want %s", expr, typ, Bool)
	}

	want := map[string]Type{
		"ref.buyer":                    Object,
		"ref.buyer.address":            Object,
		"ref.buyer.address.state":      String,
		"account_tags.user_profile":    Object,
		"account_tags.user_profile.id": Integer,
	}
	if !testutil.DeepEqual(m, want) {
		t.Errorf("Type checking %q, selector types got:\n%#v\nwant:\n%#v\n", predicate, m, want)
	}
}
