package sqlaccess

import (
	"encoding/json"
	"testing"

	"chain/testutil"
)

func TestJSONPath(t *testing.T) {
	testCases := []struct {
		json string
		path []string
		want interface{}
		err  error
	}{
		{
			json: `null`,
			path: []string{"banks", "0", "id"},
			err:  errWrongType{nil, nil},
		},
		{
			json: `5`,
			path: []string{"banks", "0", "id"},
			err:  errWrongType{nil, float64(5)},
		},
		{
			json: `"hello world"`,
			path: []string{"banks", "0", "id"},
			err:  errWrongType{nil, "hello world"},
		},
		{
			json: `{"banks": "hello world"}`,
			path: []string{"banks", "0", "id"},
			err:  errWrongType{[]string{"banks"}, "hello world"},
		},
		{
			json: `{"banks": ["hello world"]}`,
			path: []string{"banks", "0", "id"},
			err:  errWrongType{[]string{"banks", "0"}, "hello world"},
		},
		{
			json: `{"account_id": "abc123"}`,
			path: []string{"account_id"},
			want: "abc123",
		},
		{
			json: `{"exchange_rate": 1.2}`,
			path: []string{"exchange_rate"},
			want: 1.2,
		},
		{
			json: `{"account": {"number": 12345, "name": "Satoshi"}}`,
			path: []string{"account"},
			want: map[string]interface{}{"number": float64(12345), "name": "Satoshi"},
		},
		{
			json: `{"account": {"number": 12345, "name": "Satoshi"}}`,
			path: []string{"account", "number"},
			want: float64(12345),
		},
		{
			json: `{"interest_rates": [0.1, 0.2, 0.3]}`,
			path: []string{"interest_rates", "0"},
			want: 0.1,
		},
		{
			json: `{"interest_rates": [0.1, 0.2, 0.3]}`,
			path: []string{"interest_rates", "2"},
			want: 0.3,
		},
		{
			json: `{"interest_rates": [0.1, 0.2, 0.3]}`,
			path: []string{"interest_rates", "hello"},
			err: errWrongType{
				path: []string{"interest_rates"},
				val:  []interface{}{0.1, 0.2, 0.3},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(pathString(tc.path), func(t *testing.T) {
			var jsonValue interface{}
			err := json.Unmarshal([]byte(tc.json), &jsonValue)
			if err != nil {
				t.Fatal(err)
			}

			got, err := queryPath(jsonValue, tc.path...)
			if !testutil.DeepEqual(err, tc.err) {
				t.Errorf("error: got %s, want %s", err, tc.err)
			}
			if !testutil.DeepEqual(got, tc.want) {
				t.Errorf("value: got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestErrWrongType(t *testing.T) {
	testCases := []struct {
		err  errWrongType
		want string
	}{
		{
			err:  errWrongType{path: nil, val: nil},
			want: `unexpected null at root element`,
		},
		{
			err:  errWrongType{path: nil, val: 5.0},
			want: `unexpected number at root element`,
		},
		{
			err:  errWrongType{path: nil, val: []interface{}{5.0}},
			want: `unexpected array at root element`,
		},
		{
			err:  errWrongType{path: []string{"account", "creditor"}, val: 5.0},
			want: `unexpected number at "account"."creditor"`,
		},
		{
			err:  errWrongType{path: []string{"account"}, val: "hello world"},
			want: `unexpected string at "account"`,
		},
		{
			err:  errWrongType{path: []string{"account"}, val: true},
			want: `unexpected boolean at "account"`,
		},
		{
			err:  errWrongType{path: []string{"account_id"}, val: map[string]interface{}{"a": "b"}},
			want: `unexpected object at "account_id"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
