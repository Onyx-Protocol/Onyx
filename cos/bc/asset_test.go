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

	want, _ := hex.DecodeString("1bfce161790e08114f2e0897bc1975f87049618e45bdd2c57e4ed7751879dc1d")
	if !bytes.Equal(assetID[:], want) {
		t.Errorf("asset id = %x want %x", assetID[:], want)
	}
}
