package ca

import (
	//	"encoding/hex"
	"chain/crypto/ed25519/ecmath"
	"testing"
)

type testAssetIssuanceCandidate struct {
	assetID     *AssetID
	issuanceKey *ecmath.Point
}

func (candidate *testAssetIssuanceCandidate) AssetID() *AssetID {
	return candidate.assetID
}

func (candidate *testAssetIssuanceCandidate) IssuanceKey() *ecmath.Point {
	return candidate.issuanceKey
}

func TestConfidentialIARP(t *testing.T) {

	aek := []byte("asset encryption key")
	y0 := scalarHash("issuance private key 0")
	y1 := scalarHash("issuance private key 1")
	y2 := scalarHash("issuance private key 2")

	a0 := AssetID{0x00}
	a1 := AssetID{0x01}
	a2 := AssetID{0x02}

	var Y0, Y1, Y2 ecmath.Point

	Y0.ScMul(&G, &y0)
	Y1.ScMul(&G, &y1)
	Y2.ScMul(&G, &y2)

	var nonce [32]byte
	copy(nonce[:], []byte("nonce"))
	msg := []byte("message")

	secretIndex := uint64(1)
	y := y1
	ac, c := CreateAssetCommitment(a1, aek)

	candidates := []AssetIssuanceCandidate{
		&testAssetIssuanceCandidate{
			assetID:     &a0,
			issuanceKey: &Y0,
		},
		&testAssetIssuanceCandidate{
			assetID:     &a1,
			issuanceKey: &Y1,
		},
		&testAssetIssuanceCandidate{
			assetID:     &a2,
			issuanceKey: &Y2,
		},
	}

	iarp := CreateConfidentialIARP(
		ac,
		*c,
		candidates,
		nonce,
		msg,
		secretIndex,
		y,
	)

	result := iarp.Validate(
		ac,
		candidates,
		nonce,
		msg,
	)

	if result != true {
		t.Error("IARP failed to validate")
	}
}
