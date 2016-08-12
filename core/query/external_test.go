package query_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/net/context"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/blocksigner"
	"chain/core/generator"
	"chain/core/query"
	"chain/core/query/chql"
	"chain/core/txdb"
	"chain/cos"
	"chain/cos/bc"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestQueryOutputs(t *testing.T) {
	type (
		assetAccountAmount struct {
			bc.AssetAmount
			AccountID string
		}
		testcase struct {
			query  string
			values []interface{}
			when   time.Time
			want   []assetAccountAmount
		}
	)

	ctx, indexer, time1, time2, acct1, acct2, asset1, asset2 := setupQueryTest(t)

	cases := []testcase{
		{
			query:  "asset_id = $1",
			values: []interface{}{asset1.AssetID.String()},
			when:   time1,
		},
		{
			query:  "asset_tags.currency = $1",
			values: []interface{}{"USD"},
			when:   time1,
		},
		{
			query:  "asset_id = $1",
			values: []interface{}{asset1.AssetID.String()},
			when:   time2,
			want: []assetAccountAmount{
				{bc.AssetAmount{asset1.AssetID, 867}, acct1.ID},
			},
		},
		{
			query:  "asset_tags.currency = $1",
			values: []interface{}{"USD"},
			when:   time2,
			want: []assetAccountAmount{
				{bc.AssetAmount{asset1.AssetID, 867}, acct1.ID},
			},
		},
		{
			query:  "asset_id = $1",
			values: []interface{}{asset2.AssetID.String()},
			when:   time1,
		},
		{
			query:  "asset_id = $1",
			values: []interface{}{asset2.AssetID.String()},
			when:   time2,
		},
		{
			query:  "account_id = $1",
			values: []interface{}{acct1.ID},
			when:   time1,
			want:   []assetAccountAmount{},
		},
		{
			query:  "account_id = $1",
			values: []interface{}{acct1.ID},
			when:   time2,
			want: []assetAccountAmount{
				{bc.AssetAmount{asset1.AssetID, 867}, acct1.ID},
			},
		},
		{
			query:  "account_id = $1",
			values: []interface{}{acct2.ID},
			when:   time1,
			want:   []assetAccountAmount{},
		},
		{
			query:  "account_id = $1",
			values: []interface{}{acct2.ID},
			when:   time2,
			want:   []assetAccountAmount{},
		},
		{
			query:  "asset_id = $1 AND account_id = $2",
			values: []interface{}{asset1.AssetID.String(), acct1.ID},
			when:   time2,
			want: []assetAccountAmount{
				{bc.AssetAmount{asset1.AssetID, 867}, acct1.ID},
			},
		},
		{
			query:  "asset_id = $1 AND account_id = $2",
			values: []interface{}{asset2.AssetID.String(), acct1.ID},
			when:   time2,
			want:   []assetAccountAmount{},
		},
	}

	for i, tc := range cases {
		chql, err := chql.Parse(tc.query)
		if err != nil {
			t.Fatal(err)
		}
		outputs, _, err := indexer.Outputs(ctx, chql, tc.values, bc.Millis(tc.when), nil, 1000)
		if err != nil {
			t.Fatal(err)
		}
		if len(outputs) != len(tc.want) {
			t.Fatalf("case %d: got %d outputs, want %d", i, len(outputs), len(tc.want))
		}
		for j, w := range tc.want {
			var found bool
			wantAssetID := w.AssetID.String()
			for _, output := range outputs {
				got, ok := output.(map[string]interface{})
				if !ok {
					t.Fatalf("case %d: output is not a JSON object", i)
				}
				gotAssetIDItem, ok := got["asset_id"]
				if !ok {
					t.Fatalf("case %d: output does not contain asset_id", i)
				}
				gotAssetID, ok := gotAssetIDItem.(string)
				if !ok {
					t.Fatalf("case %d: output asset_id is not a string", i)
				}
				gotAmountItem, ok := got["amount"]
				if !ok {
					t.Fatalf("case %d: output does not contain amount", i)
				}
				gotAmount, ok := gotAmountItem.(float64)
				if !ok {
					t.Fatalf("case %d: output amount is not a float64", i)
				}
				gotAccountIDItem, ok := got["account_id"]
				if !ok {
					t.Fatalf("case %d: output does not contain account_id", i)
				}
				gotAccountID, ok := gotAccountIDItem.(string)
				if !ok {
					t.Fatalf("case %d: output account_id is not a string", i)
				}

				if wantAssetID == gotAssetID && w.Amount == uint64(gotAmount) && w.AccountID == gotAccountID {
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
			query  string
			values []interface{}
			when   time.Time
			want   []interface{}
		}
	)

	ctx, indexer, time1, time2, acct1, acct2, asset1, asset2 := setupQueryTest(t)

	want0 := interface{}(map[string]interface{}{"amount": uint64(0)})
	want867 := interface{}(map[string]interface{}{"amount": uint64(867)})

	cases := []testcase{
		{
			query:  "asset_id = $1",
			values: []interface{}{asset1.AssetID.String()},
			when:   time1,
			want:   []interface{}{want0},
		},
		{
			query:  "asset_tags.currency = $1",
			values: []interface{}{"USD"},
			when:   time1,
			want:   []interface{}{want0},
		},
		{
			query:  "asset_id = $1",
			values: []interface{}{asset1.AssetID.String()},
			when:   time2,
			want:   []interface{}{want867},
		},
		{
			query:  "asset_tags.currency = $1",
			values: []interface{}{"USD"},
			when:   time2,
			want:   []interface{}{want867},
		},
		{
			query:  "asset_id = $1",
			values: []interface{}{asset2.AssetID.String()},
			when:   time1,
			want:   []interface{}{want0},
		},
		{
			query:  "asset_id = $1",
			values: []interface{}{asset2.AssetID.String()},
			when:   time2,
			want:   []interface{}{want0},
		},
		{
			query:  "account_id = $1",
			values: []interface{}{acct1.ID},
			when:   time1,
			want:   []interface{}{want0},
		},
		{
			query:  "account_id = $1",
			values: []interface{}{acct1.ID},
			when:   time2,
			want:   []interface{}{want867},
		},
		{
			query:  "account_id = $1",
			values: []interface{}{acct2.ID},
			when:   time1,
			want:   []interface{}{want0},
		},
		{
			query:  "account_id = $1",
			values: []interface{}{acct2.ID},
			when:   time2,
			want:   []interface{}{want0},
		},
		{
			query:  "asset_id = $1 AND account_id = $2",
			values: []interface{}{asset1.AssetID.String(), acct1.ID},
			when:   time2,
			want:   []interface{}{want867},
		},
		{
			query:  "asset_id = $1 AND account_id = $2",
			values: []interface{}{asset2.AssetID.String(), acct1.ID},
			when:   time2,
			want:   []interface{}{want0},
		},
	}

	for i, tc := range cases {
		chql, err := chql.Parse(tc.query)
		if err != nil {
			t.Fatal(err)
		}
		balances, err := indexer.Balances(ctx, chql, tc.values, bc.Millis(tc.when))
		if err != nil {
			t.Fatal(err)
		}
		if len(balances) != len(tc.want) {
			t.Fatalf("case %d: got %d balances, want %d", i, len(balances), len(tc.want))
		}
		if !reflect.DeepEqual(balances, tc.want) {
			t.Errorf("case %d: got:\n%s\nwant:\n%s", i, spew.Sdump(balances), spew.Sdump(tc.want))
		}
	}
}

func setupQueryTest(t *testing.T) (context.Context, *query.Indexer, time.Time, time.Time, *account.Account, *account.Account, *asset.Asset, *asset.Asset) {
	time1 := time.Now()

	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	store, pool := txdb.New(db)
	fc, err := cos.NewFC(ctx, store, pool, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	indexer := query.NewIndexer(db, fc)
	asset.Init(fc, indexer, true)
	account.Init(fc)
	indexer.RegisterAnnotator(account.AnnotateTxs)
	indexer.RegisterAnnotator(asset.AnnotateTxs)
	localSigner := blocksigner.New(testutil.TestPrv, db, fc)
	g := &generator.Generator{
		Config: generator.Config{
			LocalSigner:  localSigner,
			BlockPeriod:  time.Second,
			BlockKeys:    []ed25519.PublicKey{testutil.TestPub},
			SigsRequired: 1,
			FC:           fc,
		},
	}
	genesis, err := fc.UpsertGenesisBlock(ctx, []ed25519.PublicKey{testutil.TestPub}, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	genesisHash := genesis.Hash()

	acct1, err := account.Create(ctx, []string{testutil.TestXPub.String()}, 1, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	acct2, err := account.Create(ctx, []string{testutil.TestXPub.String()}, 1, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	asset1Tags := map[string]interface{}{"currency": "USD"}

	asset1, err := asset.Define(ctx, []string{testutil.TestXPub.String()}, 1, nil, genesisHash, asset1Tags, nil)
	if err != nil {
		t.Fatal(err)
	}
	asset2, err := asset.Define(ctx, []string{testutil.TestXPub.String()}, 1, nil, genesisHash, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	assettest.IssueAssetsFixture(ctx, t, fc, asset1.AssetID, 867, acct1.ID)

	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	time2 := time.Now()

	return ctx, indexer, time1, time2, acct1, acct2, asset1, asset2
}
