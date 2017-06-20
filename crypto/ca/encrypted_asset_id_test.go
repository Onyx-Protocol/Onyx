package ca

import (
	"bytes"
	"chain/crypto/ed25519/ecmath"
	"testing"
)

func TestEncryptAssetID(t *testing.T) {
	assetID := AssetID{0xff}
	aek := []byte("asset id encryption key")

	ac, c := CreateAssetCommitment(assetID, aek)

	eaec := EncryptAssetID(ac, assetID, *c, aek)

	assetID2, c2, ok := eaec.Decrypt(ac, aek)

	if !ok {
		t.Error("decryption failed")
	} else {
		if assetID2 != assetID {
			t.Errorf("got asset ID %x, want %x", assetID2, assetID)
		}
		if !c.Equal(&c2) {
			t.Errorf("got blinding factor %x, want %x", c2[:], c[:])
		}
	}

	for i := 0; i < len(eaec.ea); i++ {
		for j := uint(0); j < 8; j++ {
			eaec.ea[i] ^= 1 << j

			assetID2, c2, ok := eaec.Decrypt(ac, aek)

			if ok {
				t.Error("unexpected decryption success with bad encrypted value amount")
			}

			if bytes.Equal(assetID[:], assetID2[:]) {
				t.Error("unexpected value from failed decryption")
			}

			if !c2.Equal(&ecmath.Zero) {
				t.Error("unexpected value from failed decryption")
			}

			eaec.ea[i] ^= 1 << j
		}
	}

	for i := 0; i < len(eaec.ec); i++ {
		for j := uint(0); j < 8; j++ {
			eaec.ec[i] ^= 1 << j

			assetID2, c2, ok := eaec.Decrypt(ac, aek)

			if ok {
				t.Error("unexpected decryption success with bad encrypted value amount")
			}

			if bytes.Equal(assetID[:], assetID2[:]) {
				t.Error("unexpected value from failed decryption")
			}

			if !c2.Equal(&ecmath.Zero) {
				t.Error("unexpected value from failed decryption")
			}

			eaec.ec[i] ^= 1 << j
		}
	}
}
