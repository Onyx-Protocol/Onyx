package bc

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestComputeAssetID(t *testing.T) {
	issuanceScript := []byte{1}
	genesisHash := mustDecodeHash("dd506f5d4c3f904d3d4b3c3be597c9198c6193ffd14a28570e4a923ce40cf9e5")
	assetID := ComputeAssetID(issuanceScript, genesisHash)

	want, _ := hex.DecodeString("9fa46bb3b31abfd5f70f187d8113bbef7ea639a5512280128d27fc3a3225b608")
	if !bytes.Equal(assetID[:], want) {
		t.Errorf("asset id = %x want %x", assetID[:], want)
	}
}
