package ca

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func (brs *borromeanRingSignature) flipBits(f func()) {
	for i := 0; i < len(brs.e); i++ {
		for j := uint(0); j < 8; j++ {
			brs.e[i] ^= 1 << j
			f()
			brs.e[i] ^= 1 << j
		}
	}
	for i := 0; i < len(brs.s); i++ {
		for j := 0; j < len(brs.s[i]); j++ {
			for k := 0; k < len(brs.s[i][j]); k++ {
				for m := uint(0); m < 8; m++ {
					brs.s[i][j][k] ^= 1 << m
					f()
					brs.s[i][j][k] ^= 1 << m
				}
			}
		}
	}
}

func Test11BRSSerialization(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))
	brs, _, err := create11BRS(msg)
	if err != nil {
		t.Fatal(err)
	}
	testBRSSerialization(t, brs, 1, 1)
}

func Test33BRSSerialization(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))
	brs, _, err := create33BRS(msg)
	if err != nil {
		t.Fatal(err)
	}
	testBRSSerialization(t, brs, 3, 3)
}

func testBRSSerialization(t *testing.T, brs *borromeanRingSignature, nRings, nPubkeys int) {
	var ser bytes.Buffer
	err := brs.writeTo(&ser)
	if err != nil {
		t.Fatal(err)
	}
	var brs2 borromeanRingSignature
	err = brs2.readFrom(bytes.NewReader(ser.Bytes()), nRings, nPubkeys)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(brs, &brs2) {
		t.Errorf("got:\n%s\nwant:\n%s", spew.Sdump(&brs2), spew.Sdump(brs))
	}
}

func Test11BRS(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))
	brs, pubkeys, err := create11BRS(msg)
	if err != nil {
		t.Fatal(err)
	}

	if err = brs.verify(msg, pubkeys); err != nil {
		t.Errorf("Borromean ring signature is not verified correctly: %s", err)
	}

	var count int

	brs.flipBits(func() {
		if count%7 == 0 {
			if err := brs.verify(msg, pubkeys); err == nil {
				t.Errorf("unexpected verification success")
			}
		}
		count++
	})
}

func Test12BRS(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))

	aliceKey := reducedScalar(hash512([]byte("alice")))
	bobKey := reducedScalar(hash512([]byte("bob")))

	alicePubkey := multiplyBasePoint(aliceKey)
	bobPubkey := multiplyBasePoint(bobKey)

	pubkeys := [][]Point{
		[]Point{
			alicePubkey,
			bobPubkey,
		},
	}

	brs, err := createBorromeanRingSignature(
		msg,
		pubkeys,
		[]Scalar{aliceKey},
		[]int{0},
		[][32]byte{
			hash256([]byte("1")),
			hash256([]byte("2")),
		},
	)

	if err != nil {
		t.Fatalf("Failed to create borromean ring signature")
	}

	if err = brs.verify(msg, pubkeys); err != nil {
		t.Errorf("Borromean ring signature is not verified correctly")
	}

	var count int

	brs.flipBits(func() {
		if count%7 == 0 {
			if err := brs.verify(msg, pubkeys); err == nil {
				t.Errorf("unexpected verification success")
			}
		}
		count++
	})
}

func Test13BRS(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))

	aliceKey := reducedScalar(hash512([]byte("alice")))
	bobKey := reducedScalar(hash512([]byte("bob")))
	carlKey := reducedScalar(hash512([]byte("carl")))

	alicePubkey := multiplyBasePoint(aliceKey)
	bobPubkey := multiplyBasePoint(bobKey)
	carlPubkey := multiplyBasePoint(carlKey)

	pubkeys := [][]Point{
		[]Point{
			alicePubkey,
			bobPubkey,
			carlPubkey,
		},
	}

	brs, err := createBorromeanRingSignature(
		msg,
		pubkeys,
		[]Scalar{aliceKey},
		[]int{0},
		[][32]byte{
			hash256([]byte("1")),
			hash256([]byte("2")),
			hash256([]byte("3")),
		},
	)

	if err != nil {
		t.Fatalf("Failed to create borromean ring signature")
	}

	if err = brs.verify(msg, pubkeys); err != nil {
		t.Errorf("Borromean ring signature is not verified correctly: %s", err)
	}

	var count int

	brs.flipBits(func() {
		if count%7 == 0 {
			if err := brs.verify(msg, pubkeys); err == nil {
				t.Errorf("unexpected verification success")
			}
		}
		count++
	})
}

func Test31BRS(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))

	aliceKey := reducedScalar(hash512([]byte("alice")))
	bobKey := reducedScalar(hash512([]byte("bob")))
	carlKey := reducedScalar(hash512([]byte("carl")))

	alicePubkey := multiplyBasePoint(aliceKey)
	bobPubkey := multiplyBasePoint(bobKey)
	carlPubkey := multiplyBasePoint(carlKey)

	pubkeys := [][]Point{
		[]Point{
			bobPubkey,
		},
		[]Point{
			carlPubkey,
		},
		[]Point{
			alicePubkey,
		},
	}

	brs, err := createBorromeanRingSignature(
		msg,
		pubkeys,
		[]Scalar{bobKey, carlKey, aliceKey},
		[]int{0, 0, 0},
		[][32]byte{
			hash256([]byte("11")),
			hash256([]byte("21")),
			hash256([]byte("31")),
		},
	)

	if err != nil {
		t.Fatalf("Failed to create borromean ring signature")
	}

	if err := brs.verify(msg, pubkeys); err != nil {
		t.Errorf("Borromean ring signature is not verified correctly: %s", err)
	}

	var count int

	brs.flipBits(func() {
		if count%7 == 0 {
			if err = brs.verify(msg, pubkeys); err == nil {
				t.Errorf("unexpected verification success")
			}
		}
		count++
	})
}

func Test33BRS(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))

	brs, pubkeys, err := create33BRS(msg)

	if err != nil {
		t.Fatalf("Failed to create borromean ring signature")
	}

	if err = brs.verify(msg, pubkeys); err != nil {
		t.Errorf("Borromean ring signature is not verified correctly")
	}

	var count int

	brs.flipBits(func() {
		if count%7 == 0 {
			if err := brs.verify(msg, pubkeys); err == nil {
				t.Errorf("unexpected verification success")
			}
		}
	})
}

func TestPayloadRecovery(t *testing.T) {
	msg := hash256([]byte("attack at dawn"))

	aliceKey := reducedScalar(hash512([]byte("alice")))
	bobKey := reducedScalar(hash512([]byte("bob")))
	carlKey := reducedScalar(hash512([]byte("carl")))

	alicePubkey := multiplyBasePoint(aliceKey)
	bobPubkey := multiplyBasePoint(bobKey)
	carlPubkey := multiplyBasePoint(carlKey)

	privkeys := []Scalar{
		aliceKey,
		carlKey,
		bobKey,
	}
	indexes := []int{2, 0, 1}
	pubkeys := [][]Point{
		[]Point{
			bobPubkey,
			carlPubkey,
			alicePubkey,
		},
		[]Point{
			carlPubkey,
			bobPubkey,
			alicePubkey,
		},
		[]Point{
			alicePubkey,
			bobPubkey,
			carlPubkey,
		},
	}

	payloadIn := [][32]byte{
		hash256([]byte{0}),
		hash256([]byte{1}),
		hash256([]byte{2}),
		hash256([]byte{3}),
		hash256([]byte{4}),
		hash256([]byte{5}),
		hash256([]byte{6}),
		hash256([]byte{7}),
		hash256([]byte{8}),
	}

	brs, err := createBorromeanRingSignature(
		msg,
		pubkeys,
		privkeys,
		indexes,
		payloadIn,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err = brs.verify(msg, pubkeys); err != nil {
		t.Error("Borromean ring signature is not verified correctly: %s", err)
	}

	payloadOut, err := brs.recoverPayload(
		msg,
		pubkeys,
		privkeys,
		indexes,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(payloadOut) != len(payloadIn) {
		t.Fatalf("Recovered payload is not the same length as the input payload: got %d, expected %d", len(payloadOut), len(payloadIn))
	}

	for i := 0; i < len(payloadOut); i++ {
		if !bytes.Equal(payloadOut[i][:], payloadIn[i][:]) {
			t.Fatalf("Recovered payload is not correct at chunk %d:\ngot %x, want %x", i, payloadOut[i][:], payloadIn[i][:])
		}
	}
}

func create11BRS(msg [32]byte) (*borromeanRingSignature, [][]Point, error) {
	aliceKey := reducedScalar(hash512([]byte("alice")))
	alicePubkey := multiplyBasePoint(aliceKey)
	pubkeys := [][]Point{
		[]Point{
			alicePubkey,
		},
	}

	brs, err := createBorromeanRingSignature(
		msg,
		pubkeys,
		[]Scalar{aliceKey},
		[]int{0},
		[][32]byte{
			hash256([]byte("1")),
		},
	)

	return brs, pubkeys, err
}

func create33BRS(msg [32]byte) (*borromeanRingSignature, [][]Point, error) {
	aliceKey := reducedScalar(hash512([]byte("alice")))
	bobKey := reducedScalar(hash512([]byte("bob")))
	carlKey := reducedScalar(hash512([]byte("carl")))

	alicePubkey := multiplyBasePoint(aliceKey)
	bobPubkey := multiplyBasePoint(bobKey)
	carlPubkey := multiplyBasePoint(carlKey)

	pubkeys := [][]Point{
		[]Point{
			bobPubkey,
			carlPubkey,
			alicePubkey,
		},
		[]Point{
			carlPubkey,
			bobPubkey,
			alicePubkey,
		},
		[]Point{
			alicePubkey,
			bobPubkey,
			carlPubkey,
		},
	}

	brs, err := createBorromeanRingSignature(
		msg,
		pubkeys,
		[]Scalar{aliceKey, carlKey, bobKey},
		[]int{2, 0, 1},
		[][32]byte{
			hash256([]byte("11")),
			hash256([]byte("12")),
			hash256([]byte("13")),
			hash256([]byte("21")),
			hash256([]byte("22")),
			hash256([]byte("23")),
			hash256([]byte("31")),
			hash256([]byte("32")),
			hash256([]byte("33")),
		},
	)
	return brs, pubkeys, err
}

func BenchmarkCreate33BRS(b *testing.B) {
	msg := hash256([]byte("attack at dawn"))
	for i := 0; i < b.N; i++ {
		_, _, err := create33BRS(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerify33BRS(b *testing.B) {
	b.StopTimer()
	msg := hash256([]byte("attack at dawn"))
	for i := 0; i < b.N; i++ {
		brs, pubkeys, err := create33BRS(msg)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if err = brs.verify(msg, pubkeys); err != nil {
			b.Errorf("unexpected BRS verification failure: %s", err)
		}
		b.StopTimer()
	}
}
