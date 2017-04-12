package query_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/generator"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/query/filter"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

func setupQueryTest(t *testing.T) (context.Context, *query.Indexer, time.Time, time.Time, string, string, bc.AssetID, bc.AssetID) {
	time1 := time.Now()

	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	indexer := query.NewIndexer(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	assets := asset.NewRegistry(db, c, pinStore)
	assets.IndexAssets(indexer)
	indexer.RegisterAnnotator(accounts.AnnotateTxs)
	indexer.RegisterAnnotator(assets.AnnotateTxs)
	go assets.ProcessBlocks(ctx)
	go accounts.ProcessBlocks(ctx)
	go indexer.ProcessBlocks(ctx)

	acct1 := coretest.CreateAccount(ctx, t, accounts, "", nil)
	acct2 := coretest.CreateAccount(ctx, t, accounts, "", nil)

	asset1Tags := map[string]interface{}{"currency": "USD", "message": "สวัสดีชาวโลก"}

	coretest.CreateAsset(ctx, t, assets, nil, "", asset1Tags)

	asset1 := coretest.CreateAsset(ctx, t, assets, nil, "", asset1Tags)
	asset2 := coretest.CreateAsset(ctx, t, assets, nil, "", nil)

	g := generator.New(c, nil, db)
	coretest.IssueAssets(ctx, t, c, g, assets, accounts, asset1, 867, acct1)
	coretest.IssueAssets(ctx, t, c, g, assets, accounts, asset2, 100, acct1)

	prottest.MakeBlock(t, c, g.PendingTxs())
	<-pinStore.PinWaiter(query.TxPinName, c.Height())

	time2 := time.Now()

	return ctx, indexer, time1, time2, acct1, acct2, asset1, asset2
}

func TestQueryOutputs(t *testing.T) {
	type (
		assetAccountAmount struct {
			bc.AssetAmount
			AccountID string
		}
		testcase struct {
			filter string
			values []interface{}
			when   time.Time
			want   []assetAccountAmount
		}
	)

	ctx, indexer, time1, time2, acct1, acct2, asset1, asset2 := setupQueryTest(t)

	cases := []testcase{
		{
			filter: "asset_id = $1",
			values: []interface{}{asset1.String()},
			when:   time1,
		},
		{
			filter: "asset_tags.currency = $1",
			values: []interface{}{"USD"},
			when:   time1,
		},
		{
			filter: "asset_id = $1",
			values: []interface{}{asset1.String()},
			when:   time2,
			want: []assetAccountAmount{
				{bc.AssetAmount{AssetId: &asset1, Amount: 867}, acct1},
			},
		},
		{
			filter: "asset_tags.currency = $1",
			values: []interface{}{"USD"},
			when:   time2,
			want: []assetAccountAmount{
				{bc.AssetAmount{AssetId: &asset1, Amount: 867}, acct1},
			},
		},
		{
			filter: "asset_tags.message = 'สวัสดีชาวโลก'",
			when:   time2,
			want: []assetAccountAmount{
				{bc.AssetAmount{AssetId: &asset1, Amount: 867}, acct1},
			},
		},
		{
			filter: "asset_tags.message = $1",
			values: []interface{}{"สวัสดีชาวโลก"},
			when:   time2,
			want: []assetAccountAmount{
				{bc.AssetAmount{AssetId: &asset1, Amount: 867}, acct1},
			},
		},
		{
			filter: "asset_id = $1",
			values: []interface{}{asset2.String()},
			when:   time1,
		},
		{
			filter: "asset_id = $1",
			values: []interface{}{asset2.String()},
			when:   time2,
			want: []assetAccountAmount{
				{bc.AssetAmount{AssetId: &asset2, Amount: 100}, acct1},
			},
		},
		{
			filter: "account_id = $1",
			values: []interface{}{acct1},
			when:   time1,
			want:   []assetAccountAmount{},
		},
		{
			filter: "account_id = $1",
			values: []interface{}{acct1},
			when:   time2,
			want: []assetAccountAmount{
				{bc.AssetAmount{AssetId: &asset2, Amount: 100}, acct1},
				{bc.AssetAmount{AssetId: &asset1, Amount: 867}, acct1},
			},
		},
		{
			filter: "account_id = $1",
			values: []interface{}{acct2},
			when:   time1,
			want:   []assetAccountAmount{},
		},
		{
			filter: "account_id = $1",
			values: []interface{}{acct2},
			when:   time2,
			want:   []assetAccountAmount{},
		},
		{
			filter: "asset_id = $1 AND account_id = $2",
			values: []interface{}{asset1.String(), acct1},
			when:   time2,
			want: []assetAccountAmount{
				{bc.AssetAmount{AssetId: &asset1, Amount: 867}, acct1},
			},
		},
		{
			filter: "asset_id = $1 AND account_id = $2",
			values: []interface{}{asset2.String(), acct2},
			when:   time2,
			want:   []assetAccountAmount{},
		},
	}

	for i, tc := range cases {
		outputs, _, err := indexer.Outputs(ctx, tc.filter, tc.values, bc.Millis(tc.when), nil, 1000)
		if err != nil {
			t.Fatal(err)
		}
		if len(outputs) != len(tc.want) {
			t.Fatalf("case %d: got %d outputs, want %d", i, len(outputs), len(tc.want))
		}
		for j, w := range tc.want {
			var found bool
			for _, output := range outputs {
				if *w.AssetId == output.AssetID && w.Amount == output.Amount && w.AccountID == output.AccountID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("case %d: did not find item %d in output", i, j)
			}
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
			values:    []interface{}{asset1.String()},
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
			values:    []interface{}{asset1.String()},
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
			values:    []interface{}{asset2.String()},
			when:      time1,
			want:      `[{"amount": 0}]`,
		},
		{
			predicate: "asset_id = $1",
			values:    []interface{}{asset2.String()},
			when:      time2,
			want:      `[{"amount": 100}]`,
		},
		{
			predicate: "account_id = $1",
			values:    []interface{}{acct1},
			when:      time1,
			want:      `[{"amount": 0}]`,
		},
		{
			predicate: "account_id = $1",
			values:    []interface{}{acct1},
			when:      time2,
			want:      `[{"amount": 967}]`,
		},
		{
			predicate: "account_id = $1",
			values:    []interface{}{acct2},
			when:      time1,
			want:      `[{"amount": 0}]`,
		},
		{
			predicate: "account_id = $1",
			values:    []interface{}{acct2},
			when:      time2,
			want:      `[{"amount": 0}]`,
		},
		{
			predicate: "asset_id = $1 AND account_id = $2",
			values:    []interface{}{asset1.String(), acct1},
			when:      time2,
			want:      `[{"amount": 867}]`,
		},
		{
			predicate: "asset_id = $1 AND account_id = $2",
			values:    []interface{}{asset2.String(), acct1},
			when:      time2,
			want:      `[{"amount": 100}]`,
		},
		{
			predicate: "asset_id = $1",
			sumBy:     []string{"account_id"},
			values:    []interface{}{asset1.String()},
			when:      time2,
			want:      `[{"sum_by": {"account_id": "` + acct1 + `"}, "amount": 867}]`,
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

		var fields []filter.Field
		for _, s := range tc.sumBy {
			f, err := filter.ParseField(s)
			if err != nil {
				t.Fatal(err)
			}
			fields = append(fields, f)
		}

		balances, err := indexer.Balances(ctx, tc.predicate, tc.values, fields, bc.Millis(tc.when))
		if err != nil {
			t.Fatal(err)
		}
		if len(balances) != len(want) {
			t.Logf("%#v", balances)
			t.Fatalf("case %d: got %d balances, want %d", i, len(balances), len(want))
		}

		got := jsonRT(t, balances)
		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got:\n%s\nwant:\n%s", i, spew.Sdump(balances), spew.Sdump(tc.want))
		}
	}
}

// jsonRT does a JSON round trip -- it marshals v
// then unmarshals the resutling JSON into an interface{}.
// This normalizes the types so it can be more easily compared
// with testutil.DeepEqual.
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
