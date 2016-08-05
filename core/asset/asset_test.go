package asset

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"golang.org/x/net/context"

	"chain/core/signers"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/testutil"
)

func TestDefineAsset(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	keys := []string{testutil.TestXPub.String()}
	var genesisHash bc.Hash
	asset, err := Define(ctx, keys, 1, nil, genesisHash, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Verify that the asset was defined.
	var id string
	var checkQ = `SELECT id FROM assets`
	err = pg.QueryRow(ctx, checkQ).Scan(&id)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if id != asset.AssetID.String() {
		t.Errorf("expected new asset %s to be recorded as %s", asset.AssetID.String(), id)
	}
}

func TestDefineAssetIdempotency(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	token := "test_token"
	keys := []string{testutil.TestXPub.String()}
	var genesisHash bc.Hash
	asset0, err := Define(ctx, keys, 1, nil, genesisHash, &token)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	asset1, err := Define(ctx, keys, 1, nil, genesisHash, &token)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// asset0 and asset1 should be exactly the same because they use the same client token
	if !reflect.DeepEqual(asset0, asset1) {
		t.Errorf("expected %v and %v to match", asset0, asset1)
	}
}

func TestDefineAndArchiveAsset(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	keys := []string{testutil.TestXPub.String()}
	var genesisHash bc.Hash
	asset, err := Define(ctx, keys, 1, nil, genesisHash, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = Archive(ctx, asset.AssetID)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// Verify that the asset was archived.
	_, err = Find(ctx, asset.AssetID)
	if err != ErrArchived {
		t.Error("expected asset id to be archived")
	}

	// Verify that the signer was archived.
	_, err = signers.Find(ctx, "asset", asset.Signer.ID)
	if errors.Root(err) != signers.ErrArchived {
		t.Error("expected signer to be archived")
	}
}

func TestFindAsset(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	keys := []string{testutil.TestXPub.String()}
	var genesisHash bc.Hash
	asset, err := Define(ctx, keys, 1, nil, genesisHash, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := Find(ctx, asset.AssetID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !reflect.DeepEqual(asset, found) {
		t.Errorf("expected %v and %v to match", asset, found)
	}
}

func TestListAsset(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	count := 3
	keys := []string{testutil.TestXPub.String()}
	var genesisHash bc.Hash

	// We do some map gymnastics because we don't know what the
	// asset ids of the generated assets will be.
	aMap := make(map[bc.AssetID]*Asset)
	for i := 0; i < count; i++ {
		a, err := Define(ctx, keys, 1, nil, genesisHash, nil)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		aMap[a.AssetID] = a
	}

	found, last, err := List(ctx, "", count)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	for i, f := range found {
		for id, asset := range aMap {
			if f.AssetID != id {
				continue
			}

			if !reflect.DeepEqual(f, asset) {
				t.Fatalf("List(ctx, \"\", 3)=found; found[%d]=%v, want %v", i, spew.Sdump(f), spew.Sdump(asset))
			}

			delete(aMap, id)
		}

		if i == len(found)-1 {
			if last != f.AssetID.String() {
				t.Errorf("`last` doesn't match last ID. Got last=%s, want %s", last, f.AssetID.String())
			}
		}
	}

	// Make sure we used up everything in aMap
	if len(aMap) != 0 {
		t.Error("Didn't find all the assets.")
	}
}

func TestAssetByClientToken(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	keys := []string{testutil.TestXPub.String()}
	token := "test_token"
	var genesisHash bc.Hash

	asset, err := Define(ctx, keys, 1, nil, genesisHash, &token)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := assetByClientToken(ctx, token)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if found.AssetID != asset.AssetID {
		t.Fatalf("assetByClientToken(\"test_token\")=%x, want %x", found.AssetID[:], asset.AssetID[:])
	}
}

func mustDecodeAssetID(hash string) bc.AssetID {
	var h bc.AssetID
	if len(hash) != hex.EncodedLen(len(h)) {
		panic("wrong length hash")
	}
	_, err := hex.Decode(h[:], []byte(hash))
	if err != nil {
		panic(err)
	}
	return h
}
