package asset_test

import (
	"encoding/hex"
	"testing"

	. "chain/api/asset"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/testutil"
)

func TestScriptDestinationPKScript(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	script := mustDecodeHex("a91400065635e652a6e00a53cfa07e822de50ccf94a887")

	dest, err := NewScriptDestination(ctx, &bc.AssetAmount{Amount: 1}, script, nil)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	got := dest.PKScript()
	testutil.ExpectScriptEqual(t, got, script, "ScriptDestination pk script")
}

func mustDecodeHex(data string) []byte {
	h, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}
	return h
}
