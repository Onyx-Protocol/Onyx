package ca

import "testing"

func TestTransientIssuanceKey(t *testing.T) {
	assetID := AssetID(fromHex256("821a9df04184630ddec3b1bab1f862d2c0a46389780368c09dcb3699dfb5b0c5"))
	aek := fromHex256("94e7bc9d0bf5faf65ecf09e3f6c9d736cea50696163b3a30eb1ff5c4d042437a")

	wanty := Scalar(fromHex256("425fa69aca0569719e86c1a1cd393e451fa72941dc68aadaa1577c3c5a34b20c"))
	wantY := mustDecodePoint(fromHex256("d29ad2472cdc0d64119946bf8fc996c4ab2949c19c64297dee8a3589ddb6a3a3"))
	y, Y := CreateTransientIssuanceKey(assetID, aek)

	if !wanty.equal(&y) {
		t.Errorf("Got %x, want %x", y, wanty)
	}
	if !wantY.equal(&Y) {
		t.Errorf("Got %x, want %x", encodePoint(&Y), encodePoint(&wantY))
	}
}
