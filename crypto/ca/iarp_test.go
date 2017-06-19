package ca

import (
	//	"encoding/hex"
	"chain/crypto/ed25519/ecmath"
	"testing"
	//	"chain/crypto/ed25519/ecmath"
)

type testAssetIssuanceKeyTuple struct {
	assetID     *AssetID
	issuanceKey IssuanceKey
}

func (tuple *testAssetIssuanceKeyTuple) AssetID() *AssetID {
	return tuple.assetID
}

func (tuple *testAssetIssuanceKeyTuple) IssuanceKey() IssuanceKey {
	return tuple.issuanceKey
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

	assetIDs := []AssetID{a0, a1, a2}
	Y := []ecmath.Point{Y0, Y1, Y2}

	iarp := CreateConfidentialIARP(
		ac,
		*c,
		assetIDs,
		Y,
		nonce,
		msg,
		secretIndex,
		y,
	)

	result := iarp.Validate(
		ac,
		assetIDs,
		Y,
		nonce,
		msg,
	)

	if result != true {
		t.Error("IARP failed to validate")
	}
}
