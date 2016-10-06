package filter

import (
	"reflect"
	"testing"
)

func TestAsSQL(t *testing.T) {
	placeholderValues := []interface{}{"foo", "bar", "baz"}
	testCases := []struct {
		q     string
		conds []interface{}
	}{
		{
		// empty predicate
		},
		{
			q:     `inputs(a = 'a' AND b = 'b')`,
			conds: []interface{}{`{"inputs":[{"a":"a","b":"b"}]}`},
		},
		{
			q:     `inputs(a = 'a') OR outputs(b = 'b')`,
			conds: []interface{}{`{"inputs":[{"a":"a"}]}`, `{"outputs":[{"b":"b"}]}`},
		},
		{
			q:     `inputs(a = 'a') AND outputs(b = 'b')`,
			conds: []interface{}{`{"inputs":[{"a":"a"}],"outputs":[{"b":"b"}]}`},
		},
		{
			q:     `inputs(a = 'a') AND inputs(b = 'b')`,
			conds: []interface{}{`{"inputs":[{"a":"a"},{"b":"b"}]}`},
		},
		{
			q:     `inputs(a = 'a') OR inputs(b = 'b')`,
			conds: []interface{}{`{"inputs":[{"a":"a"}]}`, `{"inputs":[{"b":"b"}]}`},
		},
		{
			q:     `inputs(a = 'a') AND ref.txbankref = '1ab'`,
			conds: []interface{}{`{"inputs":[{"a":"a"}],"ref":{"txbankref":"1ab"}}`},
		},
		{
			q:     `inputs(a = 'a') OR ref.txbankref = '1ab'`,
			conds: []interface{}{`{"inputs":[{"a":"a"}]}`, `{"ref":{"txbankref":"1ab"}}`},
		},
		{
			q:     `inputs(type = 'issue')`,
			conds: []interface{}{`{"inputs":[{"type":"issue"}]}`},
		},
		{
			q:     `asset_id = $3`,
			conds: []interface{}{`{"asset_id":"baz"}`},
		},
		{
			q: `inputs((a = $1 OR b = $2) AND (c = $3 OR d = 'fuzz'))`,
			conds: []interface{}{
				`{"inputs":[{"a":"foo","c":"baz"}]}`,
				`{"inputs":[{"a":"foo","d":"fuzz"}]}`,
				`{"inputs":[{"b":"bar","c":"baz"}]}`,
				`{"inputs":[{"b":"bar","d":"fuzz"}]}`,
			},
		},
		{
			q: `inputs((asset_id = 'abc' OR account_id = 'xyz') AND ref.bank_id = 'baz')`,
			conds: []interface{}{
				`{"inputs":[{"asset_id":"abc","ref":{"bank_id":"baz"}}]}`,
				`{"inputs":[{"account_id":"xyz","ref":{"bank_id":"baz"}}]}`,
			},
		},
	}

	for _, tc := range testCases {
		e, _, err := parse(tc.q)
		if err != nil {
			t.Fatal(err)
		}

		sqlExpr, err := asSQL(e, "data", placeholderValues)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(sqlExpr.Values, tc.conds) {
			t.Errorf("AsSQL(%q) = %#v, want %#v", tc.q, sqlExpr.Values, tc.conds)
		}
	}
}
