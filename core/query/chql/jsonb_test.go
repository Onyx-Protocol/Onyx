package chql

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMatchingObjects(t *testing.T) {
	placeholderValues := map[int]interface{}{
		1: "foo", 2: "bar", 3: "baz",
	}

	// test func for succinctly creating maps
	m := func(key string, v interface{}) interface{} {
		return map[string]interface{}{key: v}
	}

	testCases := []struct {
		q            string
		want         []interface{}
		wantBindings []binding
	}{
		{
			q: `INPUTS(a = 'abc' OR b = 'xyz')`,
			want: []interface{}{
				m("INPUTS", []interface{}{m("a", "abc")}),
				m("INPUTS", []interface{}{m("b", "xyz")}),
			},
		},
		{
			q: `INPUTS((a = 'abc' OR b = 'xyz') AND c = $1)`,
			want: []interface{}{
				m("INPUTS", []interface{}{map[string]interface{}{"a": "abc", "c": "foo"}}),
				m("INPUTS", []interface{}{map[string]interface{}{"b": "xyz", "c": "foo"}}),
			},
		},
		{
			q: `INPUTS(a = 'abc' AND b = 'xyz')`,
			want: []interface{}{
				m("INPUTS", []interface{}{map[string]interface{}{"a": "abc", "b": "xyz"}}),
			},
		},
		{
			q: `INPUTS(ref.recipient.email_address = 'foo@bar.com')`,
			want: []interface{}{
				m("INPUTS", []interface{}{m("ref", m("recipient", m("email_address", "foo@bar.com")))}),
			},
		},
		{
			q:    `asset_id = $1 AND account_id = $4`,
			want: []interface{}{m("asset_id", "foo")},
			wantBindings: []binding{
				{num: 4, path: []string{"account_id"}},
			},
		},
	}

	for _, tc := range testCases {
		e, _, err := parse(tc.q)
		if err != nil {
			t.Fatal(err)
		}
		got, gotBindings := matchingObjects(e, placeholderValues)
		if !reflect.DeepEqual(got, tc.want) {
			gotJSON, err := json.MarshalIndent(got, "", " ")
			if err != nil {
				t.Fatal(err)
			}
			wantJSON, err := json.MarshalIndent(tc.want, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			t.Errorf("matchingObjects(%q).conds = \n%s\n want \n%s", tc.q, gotJSON, wantJSON)
		}
		if !reflect.DeepEqual(gotBindings, tc.wantBindings) {
			t.Errorf("matchingObjects(%q).bindings = %#v, want %#v", tc.q, gotBindings, tc.wantBindings)
		}
	}
}
