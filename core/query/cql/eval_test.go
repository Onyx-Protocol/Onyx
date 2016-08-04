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
			query: `reference.recipient.email_address`,
			data:  `{"reference": {"recipient": {"id": 25356, "email_address": "hello@chain.com"}}}`,
			expected: value{
				t:   String,
				str: "hello@chain.com",
			},
		},
		{
			query: `(reference).recipient.email_address`,
			data:  `{"reference": {"recipient": {"id": 25356, "email_address": "hello@chain.com"}}}`,
			expected: value{
				t:   String,
				str: "hello@chain.com",
			},
		},
		{
			query: `reference.recipient.id`,
			data:  `{"reference": {"recipient": {"id": 25356, "email_address": "hello@chain.com"}}}`,
			expected: value{
				t:       Integer,
				integer: 25356,
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
			query: `inputs(account_tags.domestic AND account_tags.revolving)`,
			data: `{
				"inputs": [
					{ "account_tags": {"domestic": true, "priority_client": true, "revolving": false} },
					{ "account_tags": {"domestic": true, "revolving": true} }
				]
			}`,
			expected: value{t: Bool, set: Set{Invert: true}},
		},
		{
			query: `inputs(account_tags.domestic AND account_tags.revolving)`,
			data: `{
				"inputs": [
					{ "account_tags": {"revolving": false, "domestic": true, "priority_client": true} },
					{ "account_tags": {"revolving": true, "domestic": false, "international": true} }
				]
			}`,
			expected: value{t: Bool, set: Set{}},
		},
		{
			query: `NOT inputs(account_tags.domestic AND account_tags.revolving)`,
			data: `{
				"inputs": [
					{ "account_tags": {"revolving": false, "domestic": true, "priority_client": true} },
					{ "account_tags": {"revolving": true, "domestic": false, "international": true} }
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
