package bc

import (
	"testing"

	"golang.org/x/crypto/sha3"
)

func TestComputeAssetID(t *testing.T) {
	issuanceScript := []byte{1}
	initialBlockHash := mustDecodeHash("dd506f5d4c3f904d3d4b3c3be597c9198c6193ffd14a28570e4a923ce40cf9e5")
	assetID := ComputeAssetID(issuanceScript, initialBlockHash, 1, EmptyStringHash)

	unhashed := append([]byte{}, initialBlockHash[:]...)
	unhashed = append(unhashed, 0x01) // vmVersion
	unhashed = append(unhashed, 0x01) // length of issuanceScript
	unhashed = append(unhashed, issuanceScript...)
	unhashed = append(unhashed, EmptyStringHash[:]...)
	want := sha3.Sum256(unhashed)

	if assetID != want {
		t.Errorf("asset id = %x want %x", assetID[:], want[:])
	}
}

var assetIDSink AssetID

func BenchmarkComputeAssetID(b *testing.B) {
	var (
		initialBlockHash [32]byte
		issuanceScript   = []byte{5}
	)

	for i := 0; i < b.N; i++ {
		assetIDSink = ComputeAssetID(issuanceScript, initialBlockHash, 1, EmptyStringHash)
	}
}
