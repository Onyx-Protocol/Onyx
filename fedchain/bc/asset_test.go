package bc

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/txscript"

	"chain/fedchain/script"
)

func TestComputeAssetID(t *testing.T) {
	issuanceScript := script.Script{txscript.OP_1}
	genesisHash := mustDecodeHash("dd506f5d4c3f904d3d4b3c3be597c9198c6193ffd14a28570e4a923ce40cf9e5")
	assetID := ComputeAssetID(issuanceScript, genesisHash)

	want, _ := hex.DecodeString("a0f16ffd5618342611dd52589cad51f93e40cb9c54ab2e18c3169ca2e511533f")
	if !bytes.Equal(assetID[:], want) {
		t.Errorf("asset id = %x want %x", assetID[:], want)
	}
}

func TestComputeIssuanceID(t *testing.T) {
	got := ComputeIssuanceID(Outpoint{})
	want, _ := hex.DecodeString("ca5ace6dec772a290777987fd77016fcfd32925a42c84389b7b5fbd1c02654e1")
	if !bytes.Equal(got[:], want) {
		t.Errorf("issuance id = %x want %x", got[:], want)
	}
}
