package ca

import (
	"bytes"
	"reflect"
	"testing"
)

func TestAssetDescriptorSerialization(t *testing.T) {
	var assetID AssetID
	ad1 := &NonblindedAssetDescriptor{AssetID: assetID}
	var buf bytes.Buffer
	err := ad1.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	ad1a, err := ReadAssetDescriptor(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ad1, ad1a) {
		t.Errorf("got:\n%+v\nwant:\n%+v", ad1a, ad1)
	}

	H := CreateNonblindedAssetCommitment(assetID)
	ad2 := &BlindedAssetDescriptor{H: H}
	buf.Reset()
	err = ad2.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	ad2a, err := ReadAssetDescriptor(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ad2, ad2a) {
		t.Errorf("got:\n%+v\nwant:\n%+v", ad2a, ad2)
	}

	var rek RecordKey
	iek := DeriveIntermediateKey(rek)
	aek := DeriveAssetKey(iek)

	var c Scalar
	H, c = CreateBlindedAssetCommitment(H, ZeroScalar, aek)
	eaec := EncryptAssetID(assetID, H, c, aek)
	ad3 := &BlindedAssetDescriptor{H: H, eaec: &eaec}
	buf.Reset()
	err = ad3.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	ad3a, err := ReadAssetDescriptor(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ad3, ad3a) {
		t.Errorf("got:\n%+v\nwant:\n%+v", ad3a, ad3)
	}
}
