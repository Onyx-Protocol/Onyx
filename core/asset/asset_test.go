package asset

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/crypto/ed25519/chainkd"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestDefineAsset(t *testing.T) {
	r := NewRegistry(pgtest.NewTx(t), prottest.NewChain(t), nil)
	ctx := context.Background()

	keys := []chainkd.XPub{testutil.TestXPub}
	asset, err := r.Define(ctx, keys, 1, nil, "", nil, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}
	if asset.sortID == "" {
		t.Error("asset.sortID empty")
	}

	// Verify that the asset was defined.
	var id bc.AssetID
	var checkQ = `SELECT id FROM assets`
	err = r.db.QueryRowContext(ctx, checkQ).Scan(&id)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if id != asset.AssetID {
		t.Errorf("expected new asset %x to be recorded as %x", asset.AssetID.Bytes(), id.Bytes())
	}
}

func TestDefineAssetIdempotency(t *testing.T) {
	r := NewRegistry(pgtest.NewTx(t), prottest.NewChain(t), nil)
	ctx := context.Background()
	token := "test_token"
	keys := []chainkd.XPub{testutil.TestXPub}
	asset0, err := r.Define(ctx, keys, 1, nil, "alias", nil, token)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	asset1, err := r.Define(ctx, keys, 1, nil, "alias", nil, token)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// asset0 and asset1 should be exactly the same because they use the same client token
	if !testutil.DeepEqual(asset0, asset1) {
		t.Errorf("expected assets to match:\n\n%+v\n\n-----------\n\n%+v", spew.Sdump(asset0), spew.Sdump(asset1))
	}
}

func TestFindAssetByID(t *testing.T) {
	r := NewRegistry(pgtest.NewTx(t), prottest.NewChain(t), nil)
	ctx := context.Background()
	keys := []chainkd.XPub{testutil.TestXPub}
	asset, err := r.Define(ctx, keys, 1, nil, "", nil, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}
	found, err := r.findByID(ctx, asset.AssetID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !testutil.DeepEqual(asset, found) {
		t.Errorf("expected %v and %v to match", asset, found)
	}
}

func TestAssetByClientToken(t *testing.T) {
	r := NewRegistry(pgtest.NewTx(t), prottest.NewChain(t), nil)
	ctx := context.Background()
	keys := []chainkd.XPub{testutil.TestXPub}
	token := "test_token"

	asset, err := r.Define(ctx, keys, 1, nil, "", nil, token)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	found, err := assetByClientToken(ctx, r.db, token)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if found.AssetID != asset.AssetID {
		t.Fatalf("assetByClientToken(\"test_token\")=%x, want %x", found.AssetID.Bytes(), asset.AssetID.Bytes())
	}
}
