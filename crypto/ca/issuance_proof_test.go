package ca

import (
	"testing"

	"chain/crypto/ed25519/ecmath"
)

func TestIssuanceProof(t *testing.T) {
	var (
		y        [3]ecmath.Scalar // issuance private keys
		assetIDs [3]AssetID       // issuance asset ids
		Y        [3]ecmath.Point  // issuance public keys
	)
	for i := 0; i < len(assetIDs); i++ {
		y[i][0] = byte(10 + i)
		assetIDs[i][0] = byte(20 + i)
		Y[i].ScMulBase(&y[i])
	}

	var nonce [32]byte
	copy(nonce[:], []byte("nonce"))
	msg := []byte("message")

	j := uint64(1) // secret index
	aek := []byte("asset encryption key")
	ac, c := CreateAssetCommitment(assetIDs[j], aek)
	iarp := CreateConfidentialIARP(ac, *c, assetIDs[:], Y[:], nonce, msg, j, y[j])
	ip := CreateIssuanceProof(ac, iarp, assetIDs[:], msg, nonce, y[j])
	valid, yj := ip.Validate(ac, iarp, assetIDs[:], msg, nonce, j)
	if !valid {
		t.Error("failed to validate issuance proof")
	}
	if !yj {
		t.Error("validated issuance proof but not yj")
	}
}
