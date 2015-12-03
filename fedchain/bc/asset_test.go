package bc

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/txscript"
)

func TestComputeAssetID(t *testing.T) {
	issuanceScript := []byte{txscript.OP_1}
	genesisHash := mustDecodeHash("dd506f5d4c3f904d3d4b3c3be597c9198c6193ffd14a28570e4a923ce40cf9e5")
	assetID := ComputeAssetID(issuanceScript, genesisHash)

	want, _ := hex.DecodeString("a0f16ffd5618342611dd52589cad51f93e40cb9c54ab2e18c3169ca2e511533f")
	if !bytes.Equal(assetID[:], want) {
		t.Errorf("asset id = %x want %x", assetID[:], want)
	}
}
