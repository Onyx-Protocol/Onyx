package ca

// import (
// 	"bytes"
// 	"reflect"
// 	"testing"
// )

// func (ac *AssetCommitment) flipBits(f func()) {
// 	(*Point)(ac).flipBits(f)
// }

// func TestNonblindedAssetCommitment(t *testing.T) {
// 	want := fromHex("118236b5545d2ea79ccd83b43193a68843cdbcf5395d1fc03cd851d3dbdd972f")
// 	got := CreateNonblindedAssetCommitment(AssetID{})
// 	gotBytes := got.Bytes()
// 	if !bytes.Equal(gotBytes, want) {
// 		t.Errorf("Got %x, want %x", gotBytes, want)
// 	}
// }

// func TestBlindedAssetCommitment(t *testing.T) {
// 	wantH := AssetCommitment(mustDecodePoint(fromHex256("94e7bc9d0bf5faf65ecf09e3f6c9d736cea50696163b3a30eb1ff5c4d042437a")))
// 	wantd := fromHex256("4f1b5c5a3689e4d8b53d10849fc29868edc81d6dd306299ebe95860f4eb1600a")
// 	gotH, gotd := CreateBlindedAssetCommitment(
// 		AssetCommitment(mustDecodePoint(fromHex256("118236b5545d2ea79ccd83b43193a68843cdbcf5395d1fc03cd851d3dbdd972f"))),
// 		Scalar{},
// 		AssetKey(fromHex256("e8c74e3f492b4ae059e40c6966a8fe446c2e76cf2c27ccf231ba151504b42f62")),
// 	)
// 	if !reflect.DeepEqual(gotH, wantH) {
// 		t.Errorf("Got H %x, want %x", gotH.Bytes(), wantH.Bytes())
// 	}
// 	if gotd != wantd {
// 		t.Errorf("Got d %x, want %x", gotd[:], wantd[:])
// 	}
// }

// func TestAssetCommitmentOperations(t *testing.T) {
// 	for i := 0; i < 256; i++ {
// 		assetID := AssetID(hash256([]byte{byte(i)}))

// 		A := CreateNonblindedAssetCommitment(assetID)
// 		c0 := Scalar{} // initial cumulative blinding factor is zero

// 		d1 := computeDifferentialBlindingFactor(c0, hash256([]byte("1")))
// 		H1 := blindAssetCommitment(A, d1)
// 		c1 := addScalars(c0, d1)

// 		d2 := computeDifferentialBlindingFactor(c1, hash256([]byte("2")))
// 		H2 := blindAssetCommitment(H1, d2)
// 		c2 := addScalars(c1, d2)

// 		d3 := computeDifferentialBlindingFactor(c2, hash256([]byte("3")))
// 		H3 := blindAssetCommitment(H2, d3)
// 		c3 := addScalars(c2, d3)

// 		c3got := addScalars(addScalars(d1, d2), d3)
// 		if c3 != c3got {
// 			t.Errorf("Got %x, want %x", c3got[:], c3[:])
// 		}

// 		P01 := subPoints(Point(H1), Point(A))
// 		P02 := subPoints(Point(H2), Point(A))
// 		P03 := subPoints(Point(H3), Point(A))
// 		P12 := subPoints(Point(H2), Point(H1))
// 		P23 := subPoints(Point(H3), Point(H2))

// 		msg := hash256([]byte("attack at dawn"))

// 		P01_ := multiplyBasePoint(d1)
// 		if !P01_.equal(&P01) {
// 			t.Errorf("Cannot reconstruct pubkey from a differential blinding factor d1:\ngot:  %x\nwant: %x", encodePoint(&P01_), encodePoint(&P01))
// 		}

// 		sig01 := createRingSignature(msg, []Point{P01}, 0, c1)
// 		if err := sig01.verify(msg, []Point{P01}); err != nil {
// 			t.Errorf("Failed to verify signature with H1-A: %s", err)
// 		}

// 		sig02 := createRingSignature(msg, []Point{P02}, 0, c2)
// 		if err := sig02.verify(msg, []Point{P02}); err != nil {
// 			t.Errorf("Failed to verify signature with H2-A: %s", err)
// 		}

// 		sig03 := createRingSignature(msg, []Point{P03}, 0, c3)
// 		if err := sig03.verify(msg, []Point{P03}); err != nil {
// 			t.Errorf("Failed to verify signature with H3-A: %s", err)
// 		}

// 		sig12 := createRingSignature(msg, []Point{P12}, 0, d2)
// 		if err := sig12.verify(msg, []Point{P12}); err != nil {
// 			t.Errorf("Failed to verify signature with H2-H1: %s", err)
// 		}

// 		sig23 := createRingSignature(msg, []Point{P23}, 0, d3)
// 		if err := sig23.verify(msg, []Point{P23}); err != nil {
// 			t.Errorf("Failed to verify signature with H3-H2: %s", err)
// 		}
// 	}
// }
