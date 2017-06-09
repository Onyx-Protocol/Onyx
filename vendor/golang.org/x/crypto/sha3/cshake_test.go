package sha3

import (
	"encoding/hex"
	"strings"
	"testing"
)

// TODO:
// - tests for leftEncode and rightEncode - current test vectors only test a handful of values.
// - tests for cSHAKE(X,L,N=nil,S=nil) == SHAKE(X,L)

// Test vectors from http://csrc.nist.gov/groups/ST/toolkit/documents/Examples/cSHAKE_samples.pdf
func TestCShakeNISTSample1(t *testing.T) {
	shake := NewCShake128([]byte(""), []byte("Email Signature"))
	shake.Write([]byte{0x00, 0x01, 0x02, 0x03})
	output := make([]byte, 32)
	shake.Read(output)
	expected := strings.Replace("C1 C3 69 25 B6 40 9A 04 F1 B5 04 FC BC A9 D8 2B 40 17 27 7C B5 ED 2B 20 65 FC 1D 38 14 D5 AA F5", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestCShakeNISTSample1: got %s, want %s", got, expected)
	}
}

func TestCShakeNISTSample2(t *testing.T) {
	shake := NewCShake128([]byte(""), []byte("Email Signature"))
	data := make([]byte, 1600/8) // 1600 bits: "00 01 02 03 .. C4 C5 C6 C7"
	for i := byte(0); i <= 0xc7; i++ {
		data[i] = i
	}
	shake.Write(data)
	output := make([]byte, 32)
	shake.Read(output)
	expected := strings.Replace("C5 22 1D 50 E4 F8 22 D9 6A 2E 88 81 A9 61 42 0F 29 4B 7B 24 FE 3D 20 94 BA ED 2C 65 24 CC 16 6B", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestCShakeNISTSample2: got %s, want %s", got, expected)
	}
}

func TestCShakeNISTSample3(t *testing.T) {
	shake := NewCShake256([]byte(""), []byte("Email Signature"))
	shake.Write([]byte{0x00, 0x01, 0x02, 0x03})
	output := make([]byte, 64)
	shake.Read(output)
	expected := strings.Replace("D0 08 82 8E 2B 80 AC 9D 22 18 FF EE 1D 07 0C 48 B8 E4 C8 7B FF 32 C9 69 9D 5B 68 96 EE E0 ED D1 64 02 0E 2B E0 56 08 58 D9 C0 0C 03 7E 34 A9 69 37 C5 61 A7 4C 41 2B B4 C7 46 46 95 27 28 1C 8C", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestCShakeNISTSample3: got %s, want %s", got, expected)
	}
}

func TestCShakeNISTSample4(t *testing.T) {
	shake := NewCShake256([]byte(""), []byte("Email Signature"))
	data := make([]byte, 1600/8) // 1600 bits: "00 01 02 03 .. C4 C5 C6 C7"
	for i := byte(0); i <= 0xc7; i++ {
		data[i] = i
	}
	shake.Write(data)
	output := make([]byte, 64)
	shake.Read(output)
	expected := strings.Replace("07 DC 27 B1 1E 51 FB AC 75 BC 7B 3C 1D 98 3E 8B 4B 85 FB 1D EF AF 21 89 12 AC 86 43 02 73 09 17 27 F4 2B 17 ED 1D F6 3E 8E C1 18 F0 4B 23 63 3C 1D FB 15 74 C8 FB 55 CB 45 DA 8E 25 AF B0 92 BB", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestCShakeNISTSample4: got %s, want %s", got, expected)
	}
}

func BenchmarkLeftEncode(b *testing.B) {
	d := &state{rate: 104, outputLen: 48, dsbyte: 0x06}
	for i := 0; i < b.N; i++ {
		leftEncode(d, 12345)
	}
}

func BenchmarkRightEncode(b *testing.B) {
	d := &state{rate: 104, outputLen: 48, dsbyte: 0x06}
	for i := 0; i < b.N; i++ {
		rightEncode(d, 12345)
	}
}

func BenchmarkEncodeString(b *testing.B) {
	d := &state{rate: 104, outputLen: 48, dsbyte: 0x06}
	s := []byte("foo")
	for i := 0; i < b.N; i++ {
		encodeString(d, s)
	}
}

func BenchmarkInitCShake(b *testing.B) {
	d := &state{rate: 104, outputLen: 48, dsbyte: 0x06}
	N := []byte("foo")
	S := []byte("bar")
	for i := 0; i < b.N; i++ {
		d.initCShake(N, S)
	}
}
