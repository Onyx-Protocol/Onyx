package ca

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestARPSerialization(t *testing.T) {
	arp, _, _, err := createARP()
	if err != nil {
		t.Fatal(err)
	}
	var ser bytes.Buffer
	err = arp.WriteTo(&ser)
	if err != nil {
		t.Fatal(err)
	}
	var arp2 AssetRangeProof
	err = arp2.ReadFrom(bytes.NewReader(ser.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(arp, &arp2) {
		t.Errorf("got:\n%s\nwant:\n%s", spew.Sdump(&arp2), spew.Sdump(arp))
	}
}

func TestAssetRangeProof(t *testing.T) {
	rp, ac, ea, err := createARP()
	if err != nil {
		t.Fatal(err)
	}

	if err = rp.Verify(ac, &ea); err != nil {
		t.Error("failed to verify asset range proof: %s", err)
	}

	ea.AssetID[0] ^= 1
	if err = rp.Verify(ac, &ea); err == nil {
		t.Error("unexpected success from VerifyAssetRangeProof (case ea.AssetID)")
	}
	ea.AssetID[0] ^= 1

	ea.BlindingFactor[0] ^= 1
	if err = rp.Verify(ac, &ea); err == nil {
		t.Error("unexpected success from VerifyAssetRangeProof (case ea.BlindingFactor)")
	}
	ea.BlindingFactor[0] ^= 1

	for j := 0; j < 32; j++ {
		for bit := uint(0); bit < 8; bit++ {
			rp.rs.e[j] ^= byte(1 << bit)
			if err = rp.Verify(ac, &ea); err == nil {
				t.Errorf("unexpected success from VerifyAssetRangeProof (flipped the %dth bit in rp.rs.e)", j*8+int(bit))
			}
			rp.rs.e[j] ^= (1 << bit)
		}
	}
	for i := range rp.rs.s {
		for j := 0; j < 32; j++ {
			for bit := uint(0); bit < 8; bit++ {
				rp.rs.s[i][j] ^= byte(1 << bit)
				if err = rp.Verify(ac, &ea); err == nil {
					t.Errorf("unexpected success from VerifyAssetRangeProof (flipped the %dth bit in rp.rs.s[%d])", j*8+int(bit), i)
				}
				rp.rs.s[i][j] ^= (1 << bit)
			}
		}
	}
}

func createARP() (*AssetRangeProof, AssetCommitment, EncryptedAssetID, error) {
	rek := RecordKey{1}
	aek := DeriveAssetKey(DeriveIntermediateKey(rek))

	var acs []AssetCommitment

	for i := 0; i < 3; i++ {
		assetID := AssetID{byte(i)}
		nbac := CreateNonblindedAssetCommitment(assetID)
		acs = append(acs, nbac)
	}

	ac, bf := CreateBlindedAssetCommitment(acs[0], ZeroScalar, aek)
	ea := EncryptAssetID(AssetID{0}, ac, bf, aek)
	rp, err := CreateAssetRangeProof(ac, ea, acs, 0, bf)

	return rp, ac, ea, err
}
