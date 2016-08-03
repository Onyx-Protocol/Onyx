package cql

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

func TestEval(t *testing.T) {
	var testCases = []struct {
		query    string
		data     string
		expected value
	}{
		{
			query:    `'hello'`,
			expected: value{t: String, str: "hello"},
		},
		{
			query:    `'hello' = 'hello'`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query:    `'hello' = 'world'`,
			expected: value{t: Bool, set: Set{}},
		},
		{
			query:    `0xFF = 255`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query:    `0xFF != 255`,
			expected: value{t: Bool, set: Set{}},
		},
		{
			query:    `10 < 9`,
			expected: value{t: Bool, set: Set{}},
		},
		{
			query:    `'10' < '9'`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query:    `$1 = 'hello' OR account_tags CONTAINS $1`,
			data:     `{"account_tags": ["world"]}`,
			expected: value{t: Bool, set: Set{Values: []string{"hello", "world"}}},
		},
		{
			query:    `0xA >= 10`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query:    `0xA <= 10`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query:    `0xB > 10`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query:    `0xA < 10`,
			expected: value{t: Bool, set: Set{}},
		},
		{
			query:    `account_tags CONTAINS 'bank-b'`,
			data:     `{"account_tags": ["bank-a", "bank-b", "international"]}`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query: `account_tags CONTAINS $1`,
			data:  `{"account_tags": ["bank-a", "bank-b", "international"]}`,
			expected: value{
				t:   Bool,
				set: Set{Values: []string{"bank-a", "bank-b", "international"}},
			},
		},
		{
			query:    `('hello' = 'hello') = ('hello' = 'hello')`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query:    `('hello' = 'hello') != ('hello' = 'hello')`,
			expected: value{t: Bool, set: Set{}},
		},
		{
			query:    `($1 = 'hello') = ($1 = 'hello')`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query:    `($1 = 'hello') = ($1 != 'hello')`,
			expected: value{t: Bool, set: Set{}},
		},
		{
			query:    `account_tags CONTAINS $1 AND $1 != 'b'`,
			data:     `{"account_tags": ["a", "b", "c"]}`,
			expected: value{t: Bool, set: Set{Values: []string{"a", "c"}}},
		},
		{
			query:    `NOT (account_tags CONTAINS $1) AND $1 != 'c'`,
			data:     `{"account_tags": ["a", "b"]}`,
			expected: value{t: Bool, set: Set{Invert: true, Values: []string{"a", "b", "c"}}},
		},
		{
			query:    `issuance`,
			data:     `{"issuance": true}`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query:    `action = 'issue'`,
			data:     `{"action": "issue"}`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query: `inputs(account_tags CONTAINS 'domestic' AND account_tags CONTAINS 'revolving')`,
			data: `{
				"inputs": [
					{ "account_tags": ["domestic", "priority-client"] },
					{ "account_tags": ["domestic", "revolving"] }
				]
			}`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query: `inputs(account_tags CONTAINS 'domestic' AND account_tags CONTAINS 'revolving')`,
			data: `{
				"inputs": [
					{ "account_tags": ["domestic", "priority-client"] },
					{ "account_tags": ["revolving", "international"] }
				]
			}`,
			expected: value{t: Bool, set: Set{}},
		},
		{
			query: `NOT inputs(account_tags CONTAINS 'domestic' AND account_tags CONTAINS 'revolving')`,
			data: `{
				"inputs": [
					{ "account_tags": ["domestic", "priority-client"] },
					{ "account_tags": ["revolving", "international"] }
				]
			}`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
	}

	for i, tc := range testCases {
		var obj map[string]interface{}
		if tc.data != "" {
			err := json.Unmarshal([]byte(tc.data), &obj)
			if err != nil {
				t.Fatal(err)
			}
		}

		expr, _, err := parse(tc.query)
		if err != nil {
			t.Fatalf("error while parsing %s: %s", tc.query, err)
		}

		v := eval(mapEnv(obj), expr)
		sort.Strings(tc.expected.set.Values)
		sort.Strings(v.set.Values)
		if !reflect.DeepEqual(v, tc.expected) {
			t.Errorf("%d: got=%#v, want=%#v for query %s", i, v, tc.expected, expr.String())
		}
	}
}
