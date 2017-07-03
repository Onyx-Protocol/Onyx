package ca

import (
	"chain/crypto/ed25519/ecmath"
	"testing"
)

func TestNonblindedAssetIDProof(t *testing.T) {
	assetID := AssetID{1}
	ac, _ := CreateAssetCommitment(assetID, nil)
	msg := []byte("message")

	proof := CreateAssetIDProof(assetID, ac, ecmath.Zero, msg)

	if !ValidateAssetIDProof(assetID, ac, msg, proof) {
		t.Error("failed to validate asset ID proof")
	}
	if len(proof) != 0 {
		t.Error("asset ID proof for non-blinded commitment must be empty")
	}
}

func TestAssetIDProof(t *testing.T) {
	assetID := AssetID{1}
	aek := AssetKey{3}
	ac, c := CreateAssetCommitment(assetID, aek)
	msg := []byte("message")

	proof := CreateAssetIDProof(assetID, ac, *c, msg)

	if !ValidateAssetIDProof(assetID, ac, msg, proof) {
		t.Error("failed to validate asset ID proof")
	}
	if ValidateAssetIDProof(AssetID{1, 1}, ac, msg, proof) {
		t.Error("validated invalid asset ID proof")
	}
	if ValidateAssetIDProof(assetID, ac, msg[1:], proof) {
		t.Error("validated invalid asset ID proof")
	}
	proof[0] ^= 1
	if ValidateAssetIDProof(assetID, ac, msg, proof) {
		t.Error("validated invalid asset ID proof")
	}
}

func TestValueProof(t *testing.T) {
	assetID := AssetID{1}
	value := uint64(2)
	aek := AssetKey{3}
	ac, c := CreateAssetCommitment(assetID, aek)
	vek := ValueKey{4}
	vc, f := CreateValueCommitment(value, ac, vek)
	msg := []byte("message")

	vp := CreateValueProof(assetID, value, ac, vc, *c, *f, msg)

	if !vp.Validate(assetID, value, ac, vc, msg) {
		t.Error("failed to validate value proof")
	}
	if vp.Validate(AssetID{1, 1}, value, ac, vc, msg) {
		t.Error("validated invalid value proof")
	}
	if vp.Validate(assetID, value+1, ac, vc, msg) {
		t.Error("validated invalid value proof")
	}
	if vp.Validate(assetID, value, ac, vc, msg[1:]) {
		t.Error("validated invalid value proof")
	}
	vp[0] ^= 1
	if vp.Validate(assetID, value, ac, vc, msg) {
		t.Error("validated invalid value proof")
	}
}
