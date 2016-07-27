package appdb_test

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	. "chain/core/appdb"
	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestInsertManagerNode(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	newTestManagerNode(t, ctx, nil, "foo")
}

func TestAccountsWithAsset(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	_, g, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	asset1 := assettest.CreateAssetFixture(ctx, t, "", "", "")
	asset2 := assettest.CreateAssetFixture(ctx, t, "", "", "")
	mn0 := assettest.CreateManagerNodeFixture(ctx, t, "", "manager-node-0", nil, nil)
	mn1 := assettest.CreateManagerNodeFixture(ctx, t, "", "manager-node-1", nil, nil)
	acc0 := assettest.CreateAccountFixture(ctx, t, mn0, "account-0", nil)
	acc1 := assettest.CreateAccountFixture(ctx, t, mn0, "account-1", nil)
	acc2 := assettest.CreateAccountFixture(ctx, t, mn1, "account-2", nil)

	assettest.IssueAssetsFixture(ctx, t, asset1, 5, acc0)
	assettest.IssueAssetsFixture(ctx, t, asset1, 5, acc0)
	assettest.IssueAssetsFixture(ctx, t, asset1, 5, acc1)
	out1 := assettest.IssueAssetsFixture(ctx, t, asset2, 5, acc1)
	assettest.IssueAssetsFixture(ctx, t, asset1, 5, acc2)

	_, err = g.MakeBlock(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	assettest.IssueAssetsFixture(ctx, t, asset1, 1, acc0)
	out2 := assettest.IssueAssetsFixture(ctx, t, asset1, 1, acc0)
	assettest.Transfer(ctx, t, []*txbuilder.Source{
		asset.NewAccountSource(ctx, &bc.AssetAmount{AssetID: asset2, Amount: 5}, acc1, &out1.Outpoint.Hash, &out1.Outpoint.Index, nil),
		asset.NewAccountSource(ctx, &bc.AssetAmount{AssetID: asset1, Amount: 1}, acc0, &out2.Outpoint.Hash, &out2.Outpoint.Index, nil),
	}, []*txbuilder.Destination{
		txbuilder.NewScriptDestination(ctx, &bc.AssetAmount{AssetID: asset2, Amount: 5}, nil, nil),
		txbuilder.NewScriptDestination(ctx, &bc.AssetAmount{AssetID: asset1, Amount: 1}, nil, nil),
	})

	cases := []struct {
		assetID  bc.AssetID
		prev     string
		limit    int
		want     []*AccountBalanceItem
		wantLast string
	}{{
		assetID: asset1,
		prev:    "",
		limit:   50,
		want: []*AccountBalanceItem{
			{acc0, 10, 11},
			{acc1, 5, 5},
		},
		wantLast: acc1,
	}, {
		assetID: asset1,
		prev:    acc0,
		limit:   50,
		want: []*AccountBalanceItem{
			{acc1, 5, 5},
		},
		wantLast: acc1,
	}, {
		assetID: asset1,
		prev:    "",
		limit:   1,
		want: []*AccountBalanceItem{
			{acc0, 10, 11},
		},
		wantLast: acc0,
	}, {
		assetID:  asset1,
		prev:     acc1,
		limit:    50,
		want:     nil,
		wantLast: "",
	}, {
		assetID: asset2,
		prev:    "",
		limit:   50,
		want: []*AccountBalanceItem{
			{acc1, 5, 0},
		},
		wantLast: acc1,
	}}
	for _, c := range cases {
		got, gotLast, err := AccountsWithAsset(ctx, mn0, c.assetID.String(), c.prev, c.limit)
		if err != nil {
			t.Errorf("AccountsWithAsset(%q, %d) unexpected error = %q", c.prev, c.limit, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("AccountsWithAsset(%q, %d) = %+v want %+v", c.prev, c.limit, got, c.want)
		}
		if gotLast != c.wantLast {
			t.Errorf("AccountsWithAsset(%q, %d) last = %q want %q", c.prev, c.limit, gotLast, c.wantLast)
		}
	}
}
