package ca

import (
	"bytes"
	"reflect"
	"testing"
)

func TestValueDescriptorSerialization(t *testing.T) {
	var assetID AssetID
	ad1 := &NonblindedAssetDescriptor{AssetID: assetID}
	vd1 := &NonblindedValueDescriptor{Value: 1, assetDescriptor: ad1}
	var buf bytes.Buffer
	err := vd1.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	vd1a, err := ReadValueDescriptor(bytes.NewReader(buf.Bytes()), ad1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(vd1, vd1a) {
		t.Errorf("got:\n%+v\nwant:\n%+v", vd1a, vd1)
	}

	H := CreateNonblindedAssetCommitment(assetID)
	V := CreateNonblindedValueCommitment(H, 1)
	vd2 := &BlindedValueDescriptor{V: V}
	buf.Reset()
	err = vd2.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	vd2a, err := ReadValueDescriptor(bytes.NewReader(buf.Bytes()), ad1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(vd2, vd2a) {
		t.Errorf("got:\n%+v\nwant:\n%+v", vd2a, vd2)
	}

	var rek RecordKey
	iek := DeriveIntermediateKey(rek)
	aek := DeriveAssetKey(iek)
	vek := DeriveValueKey(iek)

	H, _ = CreateBlindedAssetCommitment(H, ZeroScalar, aek)

	V, f := CreateBlindedValueCommitment(vek, 1, H)
	evef := EncryptValue(V, 1, f, vek)
	vd3 := &BlindedValueDescriptor{V: V, evef: &evef}
	buf.Reset()
	err = vd3.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	vd3a, err := ReadValueDescriptor(bytes.NewReader(buf.Bytes()), ad1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(vd3, vd3a) {
		t.Errorf("got:\n%+v\nwant:\n%+v", vd3a, vd3)
	}
}
