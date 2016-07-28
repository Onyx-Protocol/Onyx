package appdb_test

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	. "chain/core/appdb"
	"chain/core/asset/assettest"
	"chain/cos/bc"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/testutil"
)

func TestAssetByID(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	ResetSeqs(ctx, t)
	xpubs := []*hd25519.XPub{testutil.TestXPub}
	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", xpubs, nil)
	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "")

	got, err := AssetByID(ctx, asset0)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	redeem, _ := hex.DecodeString("5120ca7313d5998f6005cf5a9c29677c31adfc163f599412a6ba4e9bb19d361bf4f451ae")
	issuance, _ := hex.DecodeString("76aa20d576c32879648a54df281c7839ff77a0e8315ed8fa3d34a3eb7dce22634f3d1288c0")
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
		t.Logf("got redeemscript %x, want %x", got.RedeemScript, want.RedeemScript)
		t.Logf("got issuancescript %x, want %x", got.IssuanceScript, want.IssuanceScript)
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	_, _, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)
	in1 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-1", nil, nil)
	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "def-0")
	asset1 := assettest.CreateAssetFixture(ctx, t, in0, "asset-1", "def-1")
	asset2 := assettest.CreateAssetFixture(ctx, t, in1, "asset-2", "def-2")
	asset3 := assettest.CreateAssetFixture(ctx, t, in0, "asset-3", "def-3")

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
		want    []*AssetSummary
	}{
		{
			in0,
			"",
			5,
			[]*AssetSummary{
				{ID: asset1, Label: "asset-1", Definition: def1},
				{ID: asset0, Label: "asset-0", Definition: def0},
			},
		},
		{
			in1,
			"",
			5,
			[]*AssetSummary{
				{ID: asset2, Label: "asset-2", Definition: def2},
			},
		},
		{
			in0,
			"",
			1,
			[]*AssetSummary{
				{ID: asset1, Label: "asset-1", Definition: def1},
			},
		},
		{
			in0,
			getSortID(ctx, t, asset1),
			5,
			[]*AssetSummary{
				{ID: asset0, Label: "asset-0", Definition: def0},
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	_, g, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)

	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "def-0")
	asset1 := assettest.CreateAssetFixture(ctx, t, in0, "asset-1", "def-1")

	assettest.IssueAssetsFixture(ctx, t, asset0, 58, "")

	_, err = g.MakeBlock(ctx)
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

	want := map[string]*AssetSummary{
		asset0.String(): &AssetSummary{
			ID:         asset0,
			Label:      "asset-0",
			Definition: []byte("{\n  \"s\": \"def-0\"\n}"),
		},
		asset1.String(): &AssetSummary{
			ID:         asset1,
			Label:      "asset-1",
			Definition: []byte("{\n  \"s\": \"def-1\"\n}"),
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	asset0 := assettest.CreateAssetFixture(ctx, t, "", "asset-0", "def-0")

	got, err := GetAsset(ctx, asset0.String())
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want := &AssetSummary{
		ID:         asset0,
		Label:      "asset-0",
		Definition: []byte("{\n  \"s\": \"def-0\"\n}"),
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	asset0 := assettest.CreateAssetFixture(ctx, t, "", "asset-0", "")

	assetSummary, err := GetAsset(ctx, asset0.String())
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	newLabel := "bar"
	err = UpdateAsset(ctx, assetSummary.ID.String(), &newLabel)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	assetSummary, err = GetAsset(ctx, asset0.String())
	if err != nil {
		t.Fatalf("could not get asset with id %v: %v", asset0, err)
	}
	if assetSummary.Label != newLabel {
		t.Errorf("expected %s, got %s", newLabel, assetSummary.Label)
	}
}

// Test that calling UpdateAsset with no new label is a no-op.
func TestUpdateAssetNoUpdate(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	asset0 := assettest.CreateAssetFixture(ctx, t, "", "asset-0", "")

	assetSummary, err := GetAsset(ctx, asset0.String())
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	err = UpdateAsset(ctx, assetSummary.ID.String(), nil)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	assetSummary, err = GetAsset(ctx, asset0.String())
	if err != nil {
		t.Fatalf("could not get asset with id asset-id-0: %v", err)
	}
	if assetSummary.Label != "asset-0" {
		t.Errorf("expected asset-0, got %s", assetSummary.Label)
	}
}

func TestArchiveAsset(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
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
