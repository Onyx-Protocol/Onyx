package appdb_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"golang.org/x/net/context"

	. "chain/api/appdb"
	"chain/api/asset/assettest"
	"chain/api/generator"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/testutil"
)

func TestAssetByID(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	ResetSeqs(ctx, t)
	xpubs := testutil.XPubs("xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd")
	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", xpubs, nil)
	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "")

	got, err := AssetByID(ctx, asset0)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	redeem, _ := hex.DecodeString("51210371fe1fe0352f0cea91344d06c9d9b16e394e1945ee0f3063c2f9891d163f0f5551ae")
	issuance, _ := hex.DecodeString("76a9147ca5bdd7e39cb806681d7c635b1bc36e23cbefa988c0")
	want := &Asset{
		Hash:           asset0,
		IssuerNodeID:   in0,
		INIndex:        []uint32{0, 1},
		AIndex:         []uint32{0, 0},
		Label:          "asset-0",
		RedeemScript:   redeem,
		IssuanceScript: issuance,
		Keys:           xpubs,
		Definition:     []byte("{\n  \"s\": \"\"\n}"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("asset = %#v want %#v", got, want)
	}

	// missing asset id
	_, err = AssetByID(ctx, [32]byte{1})
	if g := errors.Root(err); g != pg.ErrUserInputNotFound {
		t.Errorf("err = %v want %v", g, pg.ErrUserInputNotFound)
	}
}

func getSortID(ctx context.Context, t testing.TB, assetID bc.AssetID) (sortID string) {
	const q = `SELECT sort_id FROM assets WHERE id=$1`
	err := pg.QueryRow(ctx, q, assetID).Scan(&sortID)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return sortID
}

func TestListAssets(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	_, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)
	in1 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-1", nil, nil)

	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "def-0")
	asset1 := assettest.CreateAssetFixture(ctx, t, in0, "asset-1", "def-1")
	asset2 := assettest.CreateAssetFixture(ctx, t, in1, "asset-2", "def-2")
	asset3 := assettest.CreateAssetFixture(ctx, t, in0, "asset-3", "def-3")

	assettest.IssueAssetsFixture(ctx, t, asset0, 1, "")
	assettest.IssueAssetsFixture(ctx, t, asset1, 3, "")
	assettest.IssueAssetsFixture(ctx, t, asset2, 5, "")
	assettest.IssueAssetsFixture(ctx, t, asset3, 7, "")

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assettest.IssueAssetsFixture(ctx, t, asset0, 2, "")
	assettest.IssueAssetsFixture(ctx, t, asset1, 4, "")
	assettest.IssueAssetsFixture(ctx, t, asset2, 6, "")
	assettest.IssueAssetsFixture(ctx, t, asset3, 8, "")

	err = ArchiveAsset(ctx, asset3.String())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	def0 := []byte("{\n  \"s\": \"def-0\"\n}")
	def1 := []byte("{\n  \"s\": \"def-1\"\n}")
	def2 := []byte("{\n  \"s\": \"def-2\"\n}")

	examples := []struct {
		inodeID string
		prev    string
		limit   int
		want    []*AssetResponse
	}{
		{
			in0,
			"",
			5,
			[]*AssetResponse{
				{ID: asset1, Label: "asset-1", Issued: AssetAmount{3, 7}, Definition: def1, Circulation: 7},
				{ID: asset0, Label: "asset-0", Issued: AssetAmount{1, 3}, Definition: def0, Circulation: 3},
			},
		},
		{
			in1,
			"",
			5,
			[]*AssetResponse{
				{ID: asset2, Label: "asset-2", Issued: AssetAmount{5, 11}, Definition: def2, Circulation: 11},
			},
		},
		{
			in0,
			"",
			1,
			[]*AssetResponse{
				{ID: asset1, Label: "asset-1", Issued: AssetAmount{3, 7}, Definition: def1, Circulation: 7},
			},
		},
		{
			in0,
			getSortID(ctx, t, asset1),
			5,
			[]*AssetResponse{
				{ID: asset0, Label: "asset-0", Issued: AssetAmount{1, 3}, Definition: def0, Circulation: 3},
			},
		},
		{
			in0,
			getSortID(ctx, t, asset0),
			5,
			nil,
		},
	}

	for _, ex := range examples {
		t.Logf("ListAssets(%s, %s, %d)", ex.inodeID, ex.prev, ex.limit)

		got, _, err := ListAssets(ctx, ex.inodeID, ex.prev, ex.limit)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Fail()
			t.Log("got:")
			for _, x := range got {
				t.Logf("\t%#v", x)
			}
			t.Log("want:")
			for _, x := range ex.want {
				t.Logf("\t%#v", x)
			}
		}
	}
}

func TestGetAssets(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	_, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)

	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "def-0")
	asset1 := assettest.CreateAssetFixture(ctx, t, in0, "asset-1", "def-1")

	assettest.IssueAssetsFixture(ctx, t, asset0, 58, "")

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assettest.IssueAssetsFixture(ctx, t, asset0, 12, "")
	assettest.IssueAssetsFixture(ctx, t, asset1, 10, "")

	got, err := GetAssets(ctx, []string{
		asset0.String(),
		asset1.String(),
		"other-asset-id",
	})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want := map[string]*AssetResponse{
		asset0.String(): &AssetResponse{
			ID:          asset0,
			Label:       "asset-0",
			Definition:  []byte("{\n  \"s\": \"def-0\"\n}"),
			Issued:      AssetAmount{58, 70},
			Circulation: 70,
		},
		asset1.String(): &AssetResponse{
			ID:          asset1,
			Label:       "asset-1",
			Definition:  []byte("{\n  \"s\": \"def-1\"\n}"),
			Issued:      AssetAmount{0, 10},
			Circulation: 10,
		},
	}

	if !reflect.DeepEqual(got, want) {
		g, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			testutil.FatalErr(t, err)
		}

		w, err := json.MarshalIndent(want, "", "  ")
		if err != nil {
			testutil.FatalErr(t, err)
		}

		t.Errorf("assets:\ngot:  %v\nwant: %v", string(g), string(w))
	}
}

func TestGetAsset(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	_, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)
	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "def-0")
	assettest.IssueAssetsFixture(ctx, t, asset0, 58, "")

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assettest.IssueAssetsFixture(ctx, t, asset0, 12, "")

	got, err := GetAsset(ctx, asset0.String())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want := &AssetResponse{
		ID:          asset0,
		Label:       "asset-0",
		Definition:  []byte("{\n  \"s\": \"def-0\"\n}"),
		Issued:      AssetAmount{58, 70},
		Circulation: 70,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetAsset(%s) = %+v want %+v", asset0, got, want)
	}

	_, err = GetAsset(ctx, "nonexistent")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Errorf("GetAsset(%s) error = %q want %q", "nonexistent", errors.Root(err), pg.ErrUserInputNotFound)
	}
}

func TestUpdateAsset(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	asset0 := assettest.CreateAssetFixture(ctx, t, "", "asset-0", "")

	assetResponse, err := GetAsset(ctx, asset0.String())
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	newLabel := "bar"
	err = UpdateAsset(ctx, assetResponse.ID.String(), &newLabel)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	assetResponse, err = GetAsset(ctx, asset0.String())
	if err != nil {
		t.Fatalf("could not get asset with id %v: %v", asset0, err)
	}
	if assetResponse.Label != newLabel {
		t.Errorf("expected %s, got %s", newLabel, assetResponse.Label)
	}
}

// Test that calling UpdateAsset with no new label is a no-op.
func TestUpdateAssetNoUpdate(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	asset0 := assettest.CreateAssetFixture(ctx, t, "", "asset-0", "")

	assetResponse, err := GetAsset(ctx, asset0.String())
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	err = UpdateAsset(ctx, assetResponse.ID.String(), nil)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	assetResponse, err = GetAsset(ctx, asset0.String())
	if err != nil {
		t.Fatalf("could not get asset with id asset-id-0: %v", err)
	}
	if assetResponse.Label != "asset-0" {
		t.Errorf("expected asset-0, got %s", assetResponse.Label)
	}
}

func TestArchiveAsset(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	asset0 := assettest.CreateAssetFixture(ctx, t, "", "asset-0", "")

	err := ArchiveAsset(ctx, asset0.String())
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// Verify that the asset was archived.
	var archived bool
	var checkQ = `SELECT archived FROM assets WHERE id = $1`
	err = pg.QueryRow(ctx, checkQ, asset0.String()).Scan(&archived)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if !archived {
		t.Errorf("expected asset %s to be archived", asset0.String())
	}
}

func TestAssetBalance(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	_, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)
	mn0 := assettest.CreateManagerNodeFixture(ctx, t, "", "manager-0", nil, nil)
	mn1 := assettest.CreateManagerNodeFixture(ctx, t, "", "manager-1", nil, nil)
	acc0 := assettest.CreateAccountFixture(ctx, t, mn0, "", nil)
	acc1 := assettest.CreateAccountFixture(ctx, t, mn1, "", nil)

	var assets []bc.AssetID
	assets = append(assets, assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "def-0"))
	assets = append(assets, assettest.CreateAssetFixture(ctx, t, in0, "asset-1", "def-1"))
	assets = append(assets, assettest.CreateAssetFixture(ctx, t, in0, "asset-2", "def-2"))
	assets = append(assets, assettest.CreateAssetFixture(ctx, t, in0, "asset-3", "def-3"))
	assets = append(assets, assettest.CreateAssetFixture(ctx, t, in0, "asset-4", "def-4"))
	assets = append(assets, assettest.CreateAssetFixture(ctx, t, in0, "asset-5", "def-5"))
	sort.Sort(byAsset(assets))

	assettest.IssueAssetsFixture(ctx, t, assets[0], 1, acc0)
	assettest.IssueAssetsFixture(ctx, t, assets[0], 1, acc0)
	assettest.IssueAssetsFixture(ctx, t, assets[0], 1, acc1)
	assettest.IssueAssetsFixture(ctx, t, assets[2], 1, acc0)
	assettest.IssueAssetsFixture(ctx, t, assets[3], 1, acc0)
	assettest.IssueAssetsFixture(ctx, t, assets[5], 1, acc0)
	out1 := assettest.IssueAssetsFixture(ctx, t, assets[5], 1, acc0)
	out2 := assettest.IssueAssetsFixture(ctx, t, assets[5], 1, acc0)

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assettest.IssueAssetsFixture(ctx, t, assets[1], 1, acc0)
	out3 := assettest.IssueAssetsFixture(ctx, t, assets[1], 1, acc0)
	assettest.IssueAssetsFixture(ctx, t, assets[1], 1, acc0)
	assettest.IssueAssetsFixture(ctx, t, assets[2], 1, acc0)
	assettest.IssueAssetsFixture(ctx, t, assets[4], 1, acc0)
	assettest.IssueAssetsFixture(ctx, t, assets[4], 1, acc1)
	out4 := assettest.IssueAssetsFixture(ctx, t, assets[5], 1, acc1)

	tx := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{
			{Previous: out1.Outpoint},
			{Previous: out2.Outpoint},
			{Previous: out3.Outpoint},
			{Previous: out4.Outpoint},
		},
	})

	err = store.ApplyTx(ctx, tx, nil)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		owner     AssetOwner
		accountID string
		prev      string
		limit     int
		want      []*Balance
		wantLast  string
	}{
		{
			owner:     OwnerAccount,
			accountID: acc0,
			prev:      "",
			limit:     9999,
			want: []*Balance{
				{AssetID: assets[0], Confirmed: 2, Total: 2},
				{AssetID: assets[1], Confirmed: 0, Total: 2},
				{AssetID: assets[2], Confirmed: 1, Total: 2},
				{AssetID: assets[3], Confirmed: 1, Total: 1},
				{AssetID: assets[4], Confirmed: 0, Total: 1},
				{AssetID: assets[5], Confirmed: 3, Total: 1},
			},
			wantLast: "",
		},
		{
			owner:     OwnerAccount,
			accountID: acc0,
			prev:      "",
			limit:     1,
			want: []*Balance{
				{AssetID: assets[0], Confirmed: 2, Total: 2},
			},
			wantLast: assets[0].String(),
		},
		{
			owner:     OwnerAccount,
			accountID: acc0,
			prev:      assets[0].String(),
			limit:     1,
			want: []*Balance{
				{AssetID: assets[1], Confirmed: 0, Total: 2},
			},
			wantLast: assets[1].String(),
		},
		{
			owner:     OwnerAccount,
			accountID: acc0,
			prev:      assets[1].String(),
			limit:     1,
			want: []*Balance{
				{AssetID: assets[2], Confirmed: 1, Total: 2},
			},
			wantLast: assets[2].String(),
		},
		{
			owner:     OwnerAccount,
			accountID: acc0,
			prev:      assets[2].String(),
			limit:     1,
			want: []*Balance{
				{AssetID: assets[3], Confirmed: 1, Total: 1},
			},
			wantLast: assets[3].String(),
		},
		{
			owner:     OwnerAccount,
			accountID: acc0,
			prev:      assets[3].String(),
			limit:     1,
			want: []*Balance{
				{AssetID: assets[4], Confirmed: 0, Total: 1},
			},
			wantLast: assets[4].String(),
		},
		{
			owner:     OwnerAccount,
			accountID: acc0,
			prev:      assets[4].String(),
			limit:     1,
			want: []*Balance{
				{AssetID: assets[5], Confirmed: 3, Total: 1},
			},
			wantLast: assets[5].String(),
		},
		{
			owner:     OwnerAccount,
			accountID: acc0,
			prev:      "",
			limit:     4,
			want: []*Balance{
				{AssetID: assets[0], Confirmed: 2, Total: 2},
				{AssetID: assets[1], Confirmed: 0, Total: 2},
				{AssetID: assets[2], Confirmed: 1, Total: 2},
				{AssetID: assets[3], Confirmed: 1, Total: 1},
			},
			wantLast: assets[3].String(),
		},
		{
			owner:     OwnerAccount,
			accountID: acc0,
			prev:      assets[3].String(),
			limit:     4,
			want: []*Balance{
				{AssetID: assets[4], Confirmed: 0, Total: 1},
				{AssetID: assets[5], Confirmed: 3, Total: 1},
			},
			wantLast: "",
		},
		{
			owner:     OwnerAccount,
			accountID: acc0,
			prev:      assets[5].String(),
			limit:     4,
			want:      nil,
			wantLast:  "",
		},
		{
			owner:     OwnerAccount,
			accountID: acc1,
			prev:      "",
			limit:     9999,
			want: []*Balance{
				{AssetID: assets[0], Confirmed: 1, Total: 1},
				{AssetID: assets[4], Confirmed: 0, Total: 1},
			},
			wantLast: "",
		},

		{
			owner:     OwnerManagerNode,
			accountID: mn0,
			prev:      "",
			limit:     9999,
			want: []*Balance{
				{AssetID: assets[0], Confirmed: 2, Total: 2},
				{AssetID: assets[1], Confirmed: 0, Total: 2},
				{AssetID: assets[2], Confirmed: 1, Total: 2},
				{AssetID: assets[3], Confirmed: 1, Total: 1},
				{AssetID: assets[4], Confirmed: 0, Total: 1},
				{AssetID: assets[5], Confirmed: 3, Total: 1},
			},
			wantLast: "",
		},
		{
			owner:     OwnerManagerNode,
			accountID: mn0,
			prev:      assets[5].String(),
			limit:     9999,
			want:      nil,
			wantLast:  "",
		},
		{
			owner:     OwnerManagerNode,
			accountID: mn1,
			prev:      "",
			limit:     9999,
			want: []*Balance{
				{AssetID: assets[0], Confirmed: 1, Total: 1},
				{AssetID: assets[4], Confirmed: 0, Total: 1},
			},
			wantLast: "",
		},
		{
			owner:     OwnerManagerNode,
			accountID: mn1,
			prev:      assets[4].String(),
			limit:     9999,
			want:      nil,
			wantLast:  "",
		},
	}

	for _, c := range cases {
		got, gotLast, err := AssetBalance(ctx, &AssetBalQuery{
			Owner:   c.owner,
			OwnerID: c.accountID,
			Prev:    c.prev,
			Limit:   c.limit,
		})
		if err != nil {
			t.Errorf("AssetBalance(%s, %s, %d): unexpected error %v", c.accountID, c.prev, c.limit, err)
			continue
		}

		sort.Sort(balancesByAssetID(got))
		sort.Sort(balancesByAssetID(c.want))
		if !reflect.DeepEqual(got, c.want) {
			t.Fail()
			t.Logf("AssetBalance(%s, %s, %d)", c.accountID, c.prev, c.limit)

			t.Log("Got:")
			for _, b := range got {
				t.Log(b)
			}

			t.Log("Want:")
			for _, b := range c.want {
				t.Log(b)
			}
		}

		if gotLast != c.wantLast {
			t.Errorf("AssetBalance(%s, %s, %d) last = %v want %v", c.accountID, c.prev, c.limit, gotLast, c.wantLast)
		}
	}
}

type byAsset []bc.AssetID

func (a byAsset) Len() int           { return len(a) }
func (a byAsset) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byAsset) Less(i, j int) bool { return bytes.Compare(a[i][:], a[j][:]) < 0 }

type balancesByAssetID []*Balance

func (a balancesByAssetID) Len() int      { return len(a) }
func (a balancesByAssetID) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a balancesByAssetID) Less(i, j int) bool {
	return bytes.Compare(a[i].AssetID[:], a[j].AssetID[:]) < 0
}

func TestAccountBalanceByAssetID(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	_, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	account1 := assettest.CreateAccountFixture(ctx, t, "", "", nil)
	account2 := assettest.CreateAccountFixture(ctx, t, "", "", nil)

	var assets []bc.AssetID
	assets = append(assets, assettest.CreateAssetFixture(ctx, t, "", "asset-0", ""))
	assets = append(assets, assettest.CreateAssetFixture(ctx, t, "", "asset-1", ""))
	assets = append(assets, assettest.CreateAssetFixture(ctx, t, "", "asset-2", ""))
	assets = append(assets, assettest.CreateAssetFixture(ctx, t, "", "asset-3", ""))
	sort.Sort(byAsset(assets))

	assettest.IssueAssetsFixture(ctx, t, assets[0], 10, account1)
	assettest.IssueAssetsFixture(ctx, t, assets[0], 5, account1)
	assettest.IssueAssetsFixture(ctx, t, assets[1], 1, account1)
	assettest.IssueAssetsFixture(ctx, t, assets[2], 2, account1)
	assettest.IssueAssetsFixture(ctx, t, assets[3], 3, account2)

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	examples := []struct {
		accountID string
		assetIDs  []string
		want      []*Balance
	}{
		{
			accountID: account1,
			assetIDs: []string{
				assets[0].String(),
				assets[1].String(),
				assets[2].String(),
				assets[3].String(),
			},
			want: []*Balance{
				{AssetID: assets[0], Total: 15, Confirmed: 15},
				{AssetID: assets[1], Total: 1, Confirmed: 1},
				{AssetID: assets[2], Total: 2, Confirmed: 2},
			},
		},
		{
			accountID: account1,
			assetIDs:  []string{assets[0].String()},
			want: []*Balance{
				{AssetID: assets[0], Total: 15, Confirmed: 15},
			},
		},
		{
			accountID: account1,
			assetIDs:  []string{assets[3].String()},
			want:      nil,
		},
		{
			accountID: account2,
			assetIDs: []string{
				assets[0].String(),
				assets[1].String(),
				assets[2].String(),
				assets[3].String(),
			},
			want: []*Balance{
				{AssetID: assets[3], Total: 3, Confirmed: 3},
			},
		},
	}

	for i, ex := range examples {
		t.Log("Example", i)

		got, last, err := AssetBalance(ctx, &AssetBalQuery{
			Owner:    OwnerAccount,
			OwnerID:  ex.accountID,
			AssetIDs: ex.assetIDs,
		})
		if err != nil {
			t.Fatal("unexpected error:", err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("asset IDs:\ngot:  %v\nwant: %v", got, ex.want)
		}

		if last != "" {
			t.Errorf("got last = %q want blank", last)
		}
	}
}
