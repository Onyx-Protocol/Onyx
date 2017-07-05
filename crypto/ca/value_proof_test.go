package ca

import (
	"chain/crypto/ed25519/ecmath"
	"testing"
)

func TestNonblindedAssetIDProof(t *testing.T) {
	assetID := AssetID{1}
	ac, _ := CreateAssetCommitment(assetID, nil)
	msg := []byte("message")

	proof := ac.CreateProof(assetID, ecmath.Zero, msg)

	if !proof.Validate(assetID, ac, msg) {
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

	proof := ac.CreateProof(assetID, *c, msg)

	if !proof.Validate(assetID, ac, msg) {
		t.Error("failed to validate asset ID proof")
	}
	if proof.Validate(AssetID{1, 1}, ac, msg) {
		t.Error("validated invalid asset ID proof")
	}
	if proof.Validate(assetID, ac, msg[1:]) {
		t.Error("validated invalid asset ID proof")
	}
	proof[0] ^= 1
	if proof.Validate(assetID, ac, msg) {
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

	if !proof.Validate(value, ac, vc, msg) {
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

	if !proof.Validate(value, ac, vc, msg) {
		t.Error("failed to validate amount proof")
	}
	if proof.Validate(value+1, ac, vc, msg) {
		t.Error("validated invalid amount proof")
	}
	if proof.Validate(value, ac, vc, msg[1:]) {
		t.Error("validated invalid amount proof")
	}
	proof[0] ^= 1
	if proof.Validate(value, ac, vc, msg) {
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

	if !proof.Validate(assetID, value, ac, vc, msg) {
		t.Error("failed to validate value proof")
	}
	if proof.Validate(AssetID{1, 1}, value, ac, vc, msg) {
		t.Error("validated invalid value proof")
	}
	if proof.Validate(assetID, value+1, ac, vc, msg) {
		t.Error("validated invalid value proof")
	}
	if proof.Validate(assetID, value, ac, vc, msg[1:]) {
		t.Error("validated invalid value proof")
	}
	proof.asset[0] ^= 1
	if proof.Validate(assetID, value, ac, vc, msg) {
		t.Error("validated invalid value proof")
	}
}
