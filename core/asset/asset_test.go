package asset

import (
	"encoding/hex"
	"reflect"
	"strconv"
	"testing"

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
	asset, err := Define(ctx, keys, 1, nil, genesisHash, nil, nil)
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
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	token := "test_token"
	keys := []string{testutil.TestXPub.String()}
	var genesisHash bc.Hash
	asset0, err := Define(ctx, keys, 1, nil, genesisHash, nil, &token)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	asset1, err := Define(ctx, keys, 1, nil, genesisHash, nil, &token)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// asset0 and asset1 should be exactly the same because they use the same client token
	if !reflect.DeepEqual(asset0, asset1) {
		t.Errorf("expected %v and %v to match", asset0, asset1)
	}
}

func TestSetAssetTags(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	keys := []string{testutil.TestXPub.String()}
	var genesisHash bc.Hash
	asset, err := Define(ctx, keys, 1, nil, genesisHash, nil, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	newTags := map[string]interface{}{"someTag": "taggityTag"}

	updated, err := SetTags(ctx, asset.AssetID, newTags)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	asset.Tags = newTags
	if !reflect.DeepEqual(asset, updated) {
		t.Errorf("got = %+v want %+v", updated, asset)
	}
}

func TestSetNonLocalAssetTags(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	newTags := map[string]interface{}{"someTag": "taggityTag"}
	assetID := mustDecodeAssetID("2d194241795a28af3345ffcc64fd31d8819c56f4c4d4b4360763a259152aa393")

	updated, err := SetTags(ctx, assetID, newTags)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want := &Asset{
		AssetID: assetID,
		Tags:    newTags,
	}

	if !reflect.DeepEqual(updated, want) {
		t.Errorf("got = %+v want %+v", updated, want)
	}
}

func TestDefineAndArchiveAsset(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	keys := []string{testutil.TestXPub.String()}
	var genesisHash bc.Hash
	asset, err := Define(ctx, keys, 1, nil, genesisHash, nil, nil)
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
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	keys := []string{testutil.TestXPub.String()}
	var genesisHash bc.Hash
	asset, err := Define(ctx, keys, 1, nil, genesisHash, nil, nil)
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

func TestFindBatchAsset(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	count := 3
	keys := []string{testutil.TestXPub.String()}
	var genesisHash bc.Hash

	var assetIDs []bc.AssetID
	for i := 0; i < count; i++ {
		tags := map[string]interface{}{"number": strconv.Itoa(i)}
		a, err := Define(ctx, keys, 1, nil, genesisHash, tags, nil)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		assetIDs = append(assetIDs, a.AssetID)
	}
	t.Logf("%#v", assetIDs)

	found, err := FindBatch(ctx, assetIDs...)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	if len(found) != len(assetIDs) {
		t.Errorf("Got %d assets, want %d", len(found), len(assetIDs))
	}
}

func TestAssetByClientToken(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	keys := []string{testutil.TestXPub.String()}
	token := "test_token"
	var genesisHash bc.Hash

	asset, err := Define(ctx, keys, 1, nil, genesisHash, nil, &token)
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
