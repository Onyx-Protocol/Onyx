package ca

import (
	"bytes"
	"testing"
)

func TestEncryptOutput(t *testing.T) {
	aek := AssetKey{1}
	vek := ValueKey{2}
	idek := DataKey{3}
	assetID := AssetID{4}
	otherAssetIDs := []AssetID{AssetID{10}, AssetID{11}, AssetID{12}}
	value := uint64(5)
	plaintext := []byte("hello")
	N := uint64(8)
	eo, c, f := EncryptOutput(assetID, value, N, plaintext, nil, aek, vek, idek)

	aek0 := AssetKey{9}
	ac0, c0 := CreateAssetCommitment(assetID, aek0)
	acList := []*AssetCommitment{ac0}
	for i, other := range otherAssetIDs {
		acOther, _ := CreateAssetCommitment(other, AssetKey{byte(20 + i)})
		acList = append(acList, acOther)
	}
	arp := CreateAssetRangeProof(nil, acList, eo.ac, 0, *c0, *c) // xxx nil or msg?
	evef := EncryptValue(eo.vc, value, *f, vek)
	eaec := EncryptAssetID(eo.ac, assetID, *c, aek)
	assetID2, value2, c2, f2, plaintext2, ok := eo.Decrypt(arp, evef, eaec, aek, vek, idek)
	if !ok {
		t.Error("failed to decrypt output")
	}
	if assetID != assetID2 {
		t.Errorf("got assetID %x, want %x", assetID2[:], assetID[:])
	}
	if value != value2 {
		t.Errorf("got value %d, want %d", value2, value)
	}
	if *c != c2 {
		t.Errorf("got asset blinding factor %x, want %x", c2[:], c[:])
	}
	if *f != f2 {
		t.Errorf("got value blinding factor %x, want %x", f2[:], f[:])
	}
	if !bytes.Equal(plaintext, plaintext2) {
		t.Errorf("got plaintext %x, want %x", plaintext2, plaintext)
	}
}
