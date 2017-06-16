package ca

import "testing"

func TestVRP(t *testing.T) {
	assetID := AssetID{1}
	aek := AssetKey{2}
	ac, _ := CreateAssetCommitment(assetID, aek)

	value := uint64(3)
	vek := ValueKey{4}
	vc, f := CreateValueCommitment(value, ac, vek)

	N := uint64(8)

	pt := make([][32]byte, 2*N-1)

	idek := DataKey{5}

	msg := []byte("message")

	vrp := CreateValueRangeProof(ac, vc, N, value, pt, *f, idek, vek, msg)
	if !vrp.Validate(ac, vc, msg) {
		t.Error("failed to validate vrp")
	}
}
