package ca

import "testing"

func (e *EncryptedValue) flipBits(f func()) {
	for i := 0; i < len(e.Value); i++ {
		for j := uint(0); j < 8; j++ {
			e.Value[i] ^= 1 << j
			f()
			e.Value[i] ^= 1 << j
		}
	}
	for i := 0; i < len(e.BlindingFactor); i++ {
		for j := uint(0); j < 8; j++ {
			e.BlindingFactor[i] ^= 1 << j
			f()
			e.BlindingFactor[i] ^= 1 << j
		}
	}
}

func TestEncryptNonblindedValue(t *testing.T) {
	var assetID AssetID
	value := uint64(17)
	ac := CreateNonblindedAssetCommitment(assetID)
	vc := CreateNonblindedValueCommitment(ac, value)
	var bf Scalar
	vek := fromHex256("e8c74e3f492b4ae059e40c6966a8fe446c2e76cf2c27ccf231ba151504b42f62")
	ev := EncryptValue(vc, value, bf, vek)

	value2, bf2, ok := ev.Decrypt(vc, ac, vek)
	if !ok {
		t.Error("decryption failed")
	} else {
		if value != value2 {
			t.Errorf("got value %d, want %d", value2, value)
		}
		if bf != bf2 {
			t.Errorf("got blinding factor %x, want %x", bf2[:], bf[:])
		}
	}

	evBad1 := ev
	evBad1.Value[0] ^= 1
	_, _, ok = evBad1.Decrypt(vc, ac, vek)
	if ok {
		t.Error("unexpected decryption success with bad encrypted value amount")
	}

	evBad2 := ev
	evBad2.BlindingFactor[0] ^= 1
	_, _, ok = evBad2.Decrypt(vc, ac, vek)
	if ok {
		t.Error("unexpected decryption success with bad encrypted value blinding factor")
	}

	vekBad := vek
	vekBad[0] ^= 1
	_, _, ok = ev.Decrypt(vc, ac, vekBad)
	if ok {
		t.Error("unexpected decryption success with value encryption key")
	}
}
