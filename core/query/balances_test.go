package query

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"

	"chain/core/query/filter"
	"chain/protocol/bc"
)

func TestConstructBalancesQuery(t *testing.T) {
	now := uint64(123456)
	testCases := []struct {
		predicate  string
		sumBy      []string
		values     []interface{}
		wantQuery  string
		wantValues []interface{}
	}{
		{
			predicate:  "account_id = 'abc'",
			sumBy:      []string{"asset_id"},
			wantQuery:  `SELECT COALESCE(SUM((data->>'amount')::bigint), 0), "data"->>'asset_id' FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8 GROUP BY 2`,
			wantValues: []interface{}{`{"account_id":"abc"}`, now},
		},
		{
			predicate:  "account_id = $1",
			sumBy:      []string{"asset_id"},
			values:     []interface{}{"abc"},
			wantQuery:  `SELECT COALESCE(SUM((data->>'amount')::bigint), 0), "data"->>'asset_id' FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8 GROUP BY 2`,
			wantValues: []interface{}{`{"account_id":"abc"}`, now},
		},
		{
			predicate:  "asset_id = $1 AND account_id = $2",
			values:     []interface{}{"foo", "bar"},
			wantQuery:  `SELECT COALESCE(SUM((data->>'amount')::bigint), 0) FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8`,
			wantValues: []interface{}{`{"account_id":"bar","asset_id":"foo"}`, now},
		},
		{
			predicate:  "account_id = $1",
			sumBy:      []string{"asset_tags.currency"},
			values:     []interface{}{"foo"},
			wantQuery:  `SELECT COALESCE(SUM((data->>'amount')::bigint), 0), "data"->'asset_tags'->>'currency' FROM "annotated_outputs" WHERE ((data @> $1::jsonb)) AND timespan @> $2::int8 GROUP BY 2`,
			wantValues: []interface{}{`{"account_id":"foo"}`, now},
		},
	}

	for i, tc := range testCases {
		p, err := filter.Parse(tc.predicate)
		if err != nil {
			t.Fatal(err)
		}
		expr, err := filter.AsSQL(p, "data", tc.values)
		if err != nil {
			t.Fatal(err)
		}
		var fields []filter.Field
		for _, s := range tc.sumBy {
			f, err := filter.ParseField(s)
			if err != nil {
				t.Fatal(err)
			}
			fields = append(fields, f)
		}

		query, values := constructBalancesQuery(expr, fields, now)
		if query != tc.wantQuery {
			t.Errorf("case %d: got\n%s\nwant\n%s", i, query, tc.wantQuery)
		}
		if !reflect.DeepEqual(values, tc.wantValues) {
			t.Errorf("case %d: got %#v, want %#v", i, values, tc.wantValues)
		}
	}
}

func TestQueryBalances(t *testing.T) {
	type (
		testcase struct {
			predicate string
			sumBy     []string
			values    []interface{}
			when      time.Time
			want      string
		}
	)

	ctx, indexer, time1, time2, acct1, acct2, asset1, asset2 := setupQueryTest(t)

	cases := []testcase{
		{
			predicate: "asset_id = $1",
			values:    []interface{}{asset1.AssetID.String()},
			when:      time1,
			want:      `[{"amount": 0}]`,
		},
		{
			predicate: "asset_tags.currency = $1",
			values:    []interface{}{"USD"},
			when:      time1,
			want:      `[{"amount": 0}]`,
		},
		{
			predicate: "asset_id = $1",
			values:    []interface{}{asset1.AssetID.String()},
			when:      time2,
			want:      `[{"amount": 867}]`,
		},
		{
			predicate: "asset_tags.currency = $1",
			values:    []interface{}{"USD"},
			when:      time2,
			want:      `[{"amount": 867}]`,
		},
		{
			predicate: "asset_id = $1",
			values:    []interface{}{asset2.AssetID.String()},
			when:      time1,
			want:      `[{"amount": 0}]`,
		},
		{
			predicate: "asset_id = $1",
			values:    []interface{}{asset2.AssetID.String()},
			when:      time2,
			want:      `[{"amount": 100}]`,
		},
		{
			predicate: "account_id = $1",
			values:    []interface{}{acct1.ID},
			when:      time1,
			want:      `[{"amount": 0}]`,
		},
		{
			predicate: "account_id = $1",
			values:    []interface{}{acct1.ID},
			when:      time2,
			want:      `[{"amount": 967}]`,
		},
		{
			predicate: "account_id = $1",
			values:    []interface{}{acct2.ID},
			when:      time1,
			want:      `[{"amount": 0}]`,
		},
		{
			predicate: "account_id = $1",
			values:    []interface{}{acct2.ID},
			when:      time2,
			want:      `[{"amount": 0}]`,
		},
		{
			predicate: "asset_id = $1 AND account_id = $2",
			values:    []interface{}{asset1.AssetID.String(), acct1.ID},
			when:      time2,
			want:      `[{"amount": 867}]`,
		},
		{
			predicate: "asset_id = $1 AND account_id = $2",
			values:    []interface{}{asset2.AssetID.String(), acct1.ID},
			when:      time2,
			want:      `[{"amount": 100}]`,
		},
		{
			predicate: "asset_id = $1",
			sumBy:     []string{"account_id"},
			values:    []interface{}{asset1.AssetID.String()},
			when:      time2,
			want:      `[{"sum_by": {"account_id": "` + acct1.ID + `"}, "amount": 867}]`,
		},
		{
			sumBy: []string{"asset_tags.currency"},
			when:  time2,
			want:  `[{"sum_by": {"asset_tags.currency": "USD"}, "amount": 867}, {"sum_by": {"asset_tags.currency": null}, "amount": 100}]`,
		},
	}

	for i, tc := range cases {
		var want []interface{}
		err := json.Unmarshal([]byte(tc.want), &want)
		if err != nil {
			t.Fatal(err)
		}

		p, err := filter.Parse(tc.predicate)
		if err != nil {
			t.Fatal(err)
		}
		var fields []filter.Field
		for _, s := range tc.sumBy {
			f, err := filter.ParseField(s)
			if err != nil {
				t.Fatal(err)
			}
			fields = append(fields, f)
		}

		balances, err := indexer.Balances(ctx, p, tc.values, fields, bc.Millis(tc.when))
		if err != nil {
			t.Fatal(err)
		}
		if len(balances) != len(want) {
			t.Logf("%#v", balances)
			t.Fatalf("case %d: got %d balances, want %d", i, len(balances), len(want))
		}

		got := jsonRT(t, balances)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("case %d: got:\n%s\nwant:\n%s", i, spew.Sdump(balances), spew.Sdump(tc.want))
		}
	}
}

// jsonRT does a JSON round trip -- it marshals v
// then unmarshals the resutling JSON into an interface{}.
// This normalizes the types so it can be more easily compared
// with reflect.DeepEqual.
func jsonRT(tb testing.TB, v interface{}) interface{} {
	b, err := json.Marshal(v)
	if err != nil {
		tb.Fatal(err)
	}
	var x interface{}
	err = json.Unmarshal(b, &x)
	if err != nil {
		tb.Fatal(err)
	}
	return x
}
