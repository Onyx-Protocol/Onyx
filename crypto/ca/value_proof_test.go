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

func TestNonblindedAmountProof(t *testing.T) {
	assetID := AssetID{1}
	value := uint64(2)
	aek := AssetKey{3}
	ac, _ := CreateAssetCommitment(assetID, aek)
	vc, _ := CreateValueCommitment(value, ac, nil)
	msg := []byte("message")

	proof := CreateAmountProof(value, ac, vc, ecmath.Zero, msg)

	if !ValidateAmountProof(value, ac, vc, msg, proof) {
		t.Error("failed to validate amount proof")
	}
	if len(proof) != 0 {
		t.Error("amount proof must be empty for non-blinded value commitment")
	}
}
func TestAmountProof(t *testing.T) {
	assetID := AssetID{1}
	value := uint64(2)
	aek := AssetKey{3}
	ac, _ := CreateAssetCommitment(assetID, aek)
	vek := ValueKey{4}
	vc, f := CreateValueCommitment(value, ac, vek)
	msg := []byte("message")

	proof := CreateAmountProof(value, ac, vc, *f, msg)

	if !ValidateAmountProof(value, ac, vc, msg, proof) {
		t.Error("failed to validate amount proof")
	}
	if ValidateAmountProof(value+1, ac, vc, msg, proof) {
		t.Error("validated invalid amount proof")
	}
	if ValidateAmountProof(value, ac, vc, msg[1:], proof) {
		t.Error("validated invalid amount proof")
	}
	proof[0] ^= 1
	if ValidateAmountProof(value, ac, vc, msg, proof) {
		t.Error("validated invalid amount proof")
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

	proof := CreateValueProof(assetID, value, ac, vc, *c, *f, msg)

	if !ValidateValueProof(assetID, value, ac, vc, msg, proof) {
		t.Error("failed to validate value proof")
	}
	if ValidateValueProof(AssetID{1, 1}, value, ac, vc, msg, proof) {
		t.Error("validated invalid value proof")
	}
	if ValidateValueProof(assetID, value+1, ac, vc, msg, proof) {
		t.Error("validated invalid value proof")
	}
	if ValidateValueProof(assetID, value, ac, vc, msg[1:], proof) {
		t.Error("validated invalid value proof")
	}
	proof[0][0] ^= 1
	if ValidateValueProof(assetID, value, ac, vc, msg, proof) {
		t.Error("validated invalid value proof")
	}
}
