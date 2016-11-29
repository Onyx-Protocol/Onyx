package asset

import (
	"context"
	"reflect"
	"testing"

	"chain-stealth/core/confidentiality"
	"chain-stealth/database/pg/pgtest"
	"chain-stealth/protocol/prottest"
	"chain-stealth/testutil"
)

func TestDefineAsset(t *testing.T) {
	db := pgtest.NewTx(t)
	r := NewRegistry(db, prottest.NewChain(t), nil, &confidentiality.Storage{DB: db})
	ctx := context.Background()

	keys := []string{testutil.TestXPub.String()}
	asset, err := r.Define(ctx, keys, 1, nil, "", nil, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	if asset.sortID == "" {
		t.Error("asset.sortID empty")
	}

	// Verify that the asset was defined.
	var id string
	var checkQ = `SELECT id FROM assets`
	err = r.db.QueryRow(ctx, checkQ).Scan(&id)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if id != asset.AssetID.String() {
		t.Errorf("expected new asset %s to be recorded as %s", asset.AssetID.String(), id)
	}
}

func TestDefineAssetIdempotency(t *testing.T) {
	db := pgtest.NewTx(t)
	r := NewRegistry(db, prottest.NewChain(t), nil, &confidentiality.Storage{DB: db})
	ctx := context.Background()
	token := "test_token"
	keys := []string{testutil.TestXPub.String()}
	asset0, err := r.Define(ctx, keys, 1, nil, "", nil, &token)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	asset1, err := r.Define(ctx, keys, 1, nil, "", nil, &token)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// asset0 and asset1 should be exactly the same because they use the same client token
	if !reflect.DeepEqual(asset0, asset1) {
		t.Errorf("expected %v and %v to match", asset0, asset1)
	}
}

func TestFindAssetByID(t *testing.T) {
	db := pgtest.NewTx(t)
	r := NewRegistry(db, prottest.NewChain(t), nil, &confidentiality.Storage{DB: db})
	ctx := context.Background()
	keys := []string{testutil.TestXPub.String()}
	asset, err := r.Define(ctx, keys, 1, nil, "", nil, nil)
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
	db := pgtest.NewTx(t)
	r := NewRegistry(db, prottest.NewChain(t), nil, &confidentiality.Storage{DB: db})
	ctx := context.Background()
	keys := []string{testutil.TestXPub.String()}
	token := "test_token"

	asset, err := r.Define(ctx, keys, 1, nil, "", nil, &token)
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
