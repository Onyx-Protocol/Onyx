package ca

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func (vrp *ValueRangeProof) flipBits(f func()) {
	// xxx N, exp, vmin
	for i := range vrp.D {
		vrp.D[i].flipBits(f)
	}
	vrp.brs.flipBits(f)
}

func TestValueRangeProof(t *testing.T) {
	vrp, HC, VC, evef, err := createValueRangeProof()
	if err != nil {
		t.Fatal(err)
	}
	if err = vrp.Verify(HC, VC, &evef); err != nil {
		t.Errorf("failed to verify value range proof: %s", err)
	}

	var count int

	test := func() {
		if count%7 == 0 {
			if err := vrp.Verify(HC, VC, &evef); err == nil {
				t.Errorf("unexpected verification success")
			}
		}
		count++
	}

	vrp.flipBits(test)
	HC.flipBits(test)
	VC.flipBits(test)
	evef.flipBits(test)
}

func TestVRPSerialization(t *testing.T) {
	vrp, _, _, _, err := createValueRangeProof()
	if err != nil {
		t.Fatal(err)
	}
	var ser bytes.Buffer
	err = vrp.WriteTo(&ser)
	if err != nil {
		t.Fatal(err)
	}
	var vrp2 ValueRangeProof
	err = vrp2.ReadFrom(bytes.NewReader(ser.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(vrp, &vrp2) {
		t.Errorf("got:\n%s\n:expected:\n%s", spew.Sdump(&vrp2), spew.Sdump(vrp))
	}
}

func createValueRangeProof() (*ValueRangeProof, AssetCommitment, ValueCommitment, EncryptedValue, error) {
	var rek RecordKey
	iek := DeriveIntermediateKey(rek)
	aek := DeriveAssetKey(iek)
	vek := DeriveValueKey(iek)

	var assetID AssetID
	nonblindedAssetCommitment := CreateNonblindedAssetCommitment(assetID)
	HC, _ := CreateBlindedAssetCommitment(nonblindedAssetCommitment, Scalar{}, aek)

	value := uint64(1)

	f := Scalar{1} // value blinding factor

	VC := CreateBlindedValueCommitmentFromBlindingFactor(value, HC, f)

	evef := EncryptValue(VC, value, f, vek)

	N := uint8(8)

	pt := make([][32]byte, 2*N-1)

	vrp, err := CreateValueRangeProof(HC, VC, evef, N, value, pt, f, rek)

	if err != nil {
		return vrp, HC, VC, evef, err
	}

	ptout, err := vrp.RecoverPayload(
		HC,
		VC,
		&evef,
		value,
		f,
		rek,
	)

	if err != nil {
		return vrp, HC, VC, evef, err
	}

	if len(ptout) != len(pt) {
		return vrp, HC, VC, evef, fmt.Errorf("Recovered payload is not the same length as the input payload: got %d, expected %d", len(ptout), len(pt))
	}

	for i := 0; i < len(pt); i++ {
		if pt[i] != ptout[i] {
			return vrp, HC, VC, evef, fmt.Errorf("Recovered payload is not the same as the input payload: got %x, expected %x", ptout[i], pt[i])
		}
	}

	return vrp, HC, VC, evef, err
}

func BenchmarkCreateValueRangeProof(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _, _, err := createValueRangeProof()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerifyValueRangeProof(b *testing.B) {
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		vrp, HC, VC, evef, err := createValueRangeProof()
		if err != nil {
			b.Fatal()
		}
		b.StartTimer()
		vrp.Verify(HC, VC, &evef)
		b.StopTimer()
	}
}
