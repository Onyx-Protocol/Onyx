package filter

import (
	"errors"
	"testing"

	"chain/testutil"
)

func TestTypeCheckInvalid(t *testing.T) {
	testCases := []struct {
		p   string
		err error
	}{
		{p: `1 = 'hello world'`, err: errors.New("= expects operands of matching types")},
		{p: `inputs('hello')`, err: errors.New("inputs(...) body must have type bool")},
		{p: `inputs(1=1).bar`, err: errors.New("selector `.` can only be used on objects")},
		{p: `'hello'.foo`, err: errors.New("selector `.` can only be used on objects")},
		{p: `inputs(amount = asset_id)`, err: errors.New("= expects operands of matching types")},
		{p: `inputs(100 = asset_id)`, err: errors.New("= expects operands of matching types")},
		{p: `inputs(wat)`, err: errors.New("invalid attribute: wat")},
		{p: `wat(asset_id = 'a')`, err: errors.New("invalid environment `wat`")},
		{p: `position(asset_id = 'a')`, err: errors.New("invalid environment `position`")},
		{p: `('a' = 'a') = (1 = 1)`, err: errors.New("= expects integer or string operands")},
		{p: `1 OR 2`, err: errors.New("OR expects bool operands")},
		{p: `position.huh`, err: errors.New("selector `.` can only be used on objects")},
		{p: `ref.something = 'abc' OR ref.something = 123`, err: errors.New("\"ref.something\" used as both string and integer")},
		{p: `ref.buyer.id = 'abc' OR ref.buyer = 'hello'`, err: errors.New("\"ref.buyer\" used as both object and string")},
	}

	for _, tc := range testCases {
		expr, _, err := parse(tc.p)
		if err != nil {
			t.Fatal(err)
		}

		_, err = typeCheck(expr, transactionsSQLTable, nil)
		if !testutil.DeepEqual(err, tc.err) {
			t.Errorf("typeCheckExpr(%s) = %s, want error %s", expr, err, tc.err)
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
		{p: `$1 = '292 Ivy St'`, valTypes: []Type{String}, typ: Bool},
		{p: `'hello' = 'world'`, typ: Bool},
		{p: `id = id`, typ: Bool},
		{p: `$1 = 'hello' OR ref.something = $1`, valTypes: []Type{String}, typ: Bool},
		{p: `($1 = 'hello') OR (ref.something = $1)`, valTypes: []Type{String}, typ: Bool},
		{p: `inputs(account_tags.domestic AND account_tags.type = 'revolving')`, typ: Bool},
		{p: `inputs(account_tags.state = account_tags.shipping_address.state)`, typ: Bool},
		{p: `inputs(account_tags.a_boolean_field)`, typ: Bool},
		{p: `ref.a_boolean_field AND ref.another_boolean_field`, typ: Bool},
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
