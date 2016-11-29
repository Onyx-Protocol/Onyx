package ca

import (
	"bytes"
	"testing"
)

func TestEmptyPayload(t *testing.T) {
	ek := hash256([]byte("encryption key"))

	plaintext := [][32]byte{}
	ciphertext := EncryptPayload(plaintext, ek)

	if len(ciphertext) != 1 {
		t.Fatalf("EncryptPayload(empty) should yield 1 chunk containing MAC, got %d", len(ciphertext))
	}

	mac := hash256(ek[:])
	want := mac[:]
	got := serializePayload(ciphertext)
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}

	want = fromHex("664fb7c63f54da096fe1532aeaee986385f10623758c28b4dc27a466df7a0b7c")
	got = serializePayload(ciphertext)
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}

	pt, ok := DecryptPayload(ciphertext, ek)

	if !ok {
		t.Fatal("Failed to authenticate ciphertext")
	}

	want = fromHex("")
	got = serializePayload(pt)
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}
}

func Test1ItemPayload(t *testing.T) {
	ek := hash256([]byte("encryption key"))

	plaintext := [][32]byte{[32]byte{0x01, 0x02, 0x03, 0x04}}
	ciphertext := EncryptPayload(plaintext, ek)

	if len(ciphertext) != 2 {
		t.Fatalf("EncryptPayload('0102030400000...') should yield 2 chunks containing ciphertext and MAC, got %d", len(ciphertext))
	}

	want := fromHex("7da0aba41231b699cd8fad9d2680e4ef4e8bacb7f3a77bbec398f05709e45d5d" +
		"8f71c930d6935c836e985f1085778e797294ddb54475b5d0a2cdf1c982ce4056")
	got := serializePayload(ciphertext)
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}

	pt, ok := DecryptPayload(ciphertext, ek)
	if !ok {
		t.Fatalf("Failed to authenticate ciphertext")
	}
	want = fromHex("0102030400000000000000000000000000000000000000000000000000000000")
	got = serializePayload(pt)
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}
}

func Test3ItemPayload(t *testing.T) {
	ek := hash256([]byte("encryption key"))

	plaintext := [][32]byte{[32]byte{0x01}, [32]byte{0x02}, [32]byte{0x03}}
	ciphertext := EncryptPayload(plaintext, ek)

	if len(ciphertext) != 4 {
		t.Fatalf("EncryptPayload('01000000...0200000....0300000....') should yield 4 chunks containing ciphertext and MAC")
	}

	want := fromHex("7da2a8a01231b699cd8fad9d2680e4ef4e8bacb7f3a77bbec398f05709e45d5d" +
		"5507e88e0fb486b8ca1bfbac8846048f3d776596a1a8da0891329d65343e9373" +
		"109c82a3317a1e123881d339ead80576e6294040f550a3715863be3aa0ac6ca2" +
		"18c400d809d6bc9ea0b7b80c0815803842281000f305f7e8f875d56032e0a925")
	got := serializePayload(ciphertext)
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}

	pt, ok := DecryptPayload(ciphertext, ek)
	if !ok {
		t.Fatalf("Failed to authenticate ciphertext")
	}
	want = fromHex("0100000000000000000000000000000000000000000000000000000000000000" +
		"0200000000000000000000000000000000000000000000000000000000000000" +
		"0300000000000000000000000000000000000000000000000000000000000000")
	got = serializePayload(pt)
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}
}

func serializePayload(input [][32]byte) []byte {
	buf := make([]byte, 0, len(input)*32)
	for _, chunk := range input {
		for i := 0; i < 32; i++ {
			buf = append(buf, chunk[i])
		}
	}
	return buf
}
