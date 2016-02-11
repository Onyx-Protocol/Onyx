package txbuilder

import (
	"testing"

	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/testutil"
)

func TestScriptDestinationPKScript(t *testing.T) {
	ctx := context.Background()

	script := mustDecodeHex("a91400065635e652a6e00a53cfa07e822de50ccf94a887")

	dest := NewScriptDestination(ctx, &bc.AssetAmount{Amount: 1}, script, nil)
	got := dest.PKScript()
	testutil.ExpectScriptEqual(t, got, script, "ScriptDestination pk script")
}
