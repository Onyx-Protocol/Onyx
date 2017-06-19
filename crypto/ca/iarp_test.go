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

	nonce := []byte("nonce")
	msg := []byte("message")

	secretIndex := uint64(1)
	y := y1
	ac, c := CreateAssetCommitment(a1, aek)

	keyTuples := []AssetIssuanceKeyTuple{
		&testAssetIssuanceKeyTuple{
			assetID:     &a0,
			issuanceKey: Y0.Bytes(),
		},
		&testAssetIssuanceKeyTuple{
			assetID:     &a1,
			issuanceKey: Y1.Bytes(),
		},
		&testAssetIssuanceKeyTuple{
			assetID:     &a2,
			issuanceKey: Y2.Bytes(),
		},
	}

	iarp := CreateConfidentialIARP(
		ac,
		*c,
		keyTuples,
		nonce,
		msg,
		secretIndex,
		y,
	)

	result := iarp.Validate(
		ac,
		keyTuples,
		nonce,
		msg,
	)

	if result != true {
		t.Error("IARP failed to validate")
	}
}
