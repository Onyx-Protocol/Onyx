package ca

import (
	"bytes"
	"testing"
)

func TestEncryptIssuance(t *testing.T) {
	var rek RecordKey
	assetID := AssetID{1}
	value := uint64(2)
	N := uint8(8)
	assetIDs := []AssetID{{3}, {4}, assetID, {5}}
	j := 2
	iek := DeriveIntermediateKey(rek)
	aek := DeriveAssetKey(iek)
	Y := make([]Point, len(assetIDs))
	var y Scalar
	for i, assetID := range assetIDs {
		var priv Scalar
		priv, Y[i] = CreateTransientIssuanceKey(assetID, aek)
		if i == j {
			y = priv
		}
	}
	vmver := uint64(2)
	program := []byte{0x51} // OP_TRUE, minus the dependency on protocol/vm

	AD, VD, iarp, vrp, _, _, err := EncryptIssuance(rek, assetID, value, N, assetIDs, Y, y, vmver, program)
	if err != nil {
		t.Fatal(err)
	}
	err = VerifyIssuance(issuanceStruct{
		AD:   AD,
		VD:   VD,
		AIDs: assetIDs,
		IARP: iarp,
		VRP:  vrp,
	})
	if err != nil {
		t.Errorf("Issuance verification failed: %s", err)
	}
}

func TestEncryptDecryptOutput(t *testing.T) {
	var rek RecordKey

	iek := DeriveIntermediateKey(rek)
	aek := DeriveAssetKey(iek)

	assetID := AssetID{1}
	amount := uint64(2)

	N := uint8(8)
	a := []AssetID{{3}, {4}, assetID, {5}}
	H := make([]AssetCommitment, len(a))
	var cprev Scalar
	for i, assetID := range a {
		ac := CreateNonblindedAssetCommitment(assetID)
		bf := ZeroScalar
		d := ZeroScalar
		for k := 0; k < i; k++ {
			ac, d = CreateBlindedAssetCommitment(ac, bf, aek)
			bf.Add(&d)
		}
		H[i] = ac
		if i == 2 {
			cprev = bf
		}
	}
	plaintext := []byte("foo")

	AD, VD, arp, vrp, c, f, err := EncryptOutput(rek, assetID, amount, N, H, cprev, plaintext, nil)
	_ = c
	_ = f
	_ = arp

	if err != nil {
		t.Fatal(err)
	}

	assetID2, amount2, _, _, plaintext2, err := DecryptOutput(rek, AD, VD, vrp)
	if err != nil {
		t.Fatal(err)
	}

	if assetID != assetID2 || amount != amount2 {
		t.Errorf("after decryption, assetamounts disagree: got %x/%d, want %x/%d", assetID[:], amount, assetID2[:], amount)
	}

	if !bytes.Equal(plaintext, plaintext2) {
		t.Errorf("after decryption, plaintexts disagree: got %x, want %x", plaintext2, plaintext)
	}
}
