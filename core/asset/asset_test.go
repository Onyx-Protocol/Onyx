package asset

import (
	"bytes"
	"context"
	"reflect"
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
	err = r.db.QueryRow(ctx, checkQ).Scan(&id)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if !bytes.Equal(id[:], asset.AssetID[:]) {
		t.Errorf("expected new asset %s to be recorded as %s", asset.AssetID, id)
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
	if !reflect.DeepEqual(asset0, asset1) {
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

	if !reflect.DeepEqual(asset, found) {
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
		t.Fatalf("assetByClientToken(\"test_token\")=%x, want %x", found.AssetID[:], asset.AssetID[:])
	}
}
