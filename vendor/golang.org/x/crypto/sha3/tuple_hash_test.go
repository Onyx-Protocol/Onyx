package sha3

import (
	"encoding/hex"
	"strings"
	"testing"
)

// Test vectors from:
// - http://csrc.nist.gov/groups/ST/toolkit/documents/Examples/TupleHash_samples.pdf
// - http://csrc.nist.gov/groups/ST/toolkit/documents/Examples/TupleHashXOF_samples.pdf

func TestTupleHashNISTSample1(t *testing.T) {
	outputLength := 32
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
	}
	S := []byte{}
	output := make([]byte, outputLength)
	TupleHash128(tuples, S, output)
	expected := strings.Replace("C5 D8 78 6C 1A FB 9B 82 11 1A B3 4B 65 B2 C0 04 8F A6 4E 6D 48 E2 63 26 4C E1 70 7D 3F FC 8E D1", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample1: got %s, want %s", got, expected)
	}

	h := NewTupleHash128(outputLength, S)
	for _, item := range tuples {
		h.Write(item)
	}
	copy(output, zero[:])
	h.Sum(output[:0])
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample1: got %s, want %s", got, expected)
	}
}

func TestTupleHashNISTSample2(t *testing.T) {
	outputLength := 32
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
	}
	S := []byte("My Tuple App")
	output := make([]byte, outputLength)
	TupleHash128(tuples, S, output)
	expected := strings.Replace("75 CD B2 0F F4 DB 11 54 E8 41 D7 58 E2 41 60 C5 4B AE 86 EB 8C 13 E7 F5 F4 0E B3 55 88 E9 6D FB", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample2: got %s, want %s", got, expected)
	}

	h := NewTupleHash128(outputLength, S)
	for _, item := range tuples {
		h.Write(item)
	}
	copy(output, zero[:])
	h.Sum(output[:0])
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample2: got %s, want %s", got, expected)
	}
}

func TestTupleHashNISTSample3(t *testing.T) {
	outputLength := 32
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
		[]byte{0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28},
	}
	S := []byte("My Tuple App")
	output := make([]byte, outputLength)
	TupleHash128(tuples, S, output)
	expected := strings.Replace("E6 0F 20 2C 89 A2 63 1E DA 8D 4C 58 8C A5 FD 07 F3 9E 51 51 99 8D EC CF 97 3A DB 38 04 BB 6E 84", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample3: got %s, want %s", got, expected)
	}

	h := NewTupleHash128(outputLength, S)
	for _, item := range tuples {
		h.Write(item)
	}
	copy(output, zero[:])
	h.Sum(output[:0])
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample3: got %s, want %s", got, expected)
	}
}

func TestTupleHashNISTSample4(t *testing.T) {
	outputLength := 64
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
	}
	S := []byte{}
	output := make([]byte, outputLength)
	TupleHash256(tuples, S, output)
	expected := strings.Replace("CF B7 05 8C AC A5 E6 68 F8 1A 12 A2 0A 21 95 CE 97 A9 25 F1 DB A3 E7 44 9A 56 F8 22 01 EC 60 73 11 AC 26 96 B1 AB 5E A2 35 2D F1 42 3B DE 7B D4 BB 78 C9 AE D1 A8 53 C7 86 72 F9 EB 23 BB E1 94", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample4: got %s, want %s", got, expected)
	}

	h := NewTupleHash256(outputLength, S)
	for _, item := range tuples {
		h.Write(item)
	}
	copy(output, zero[:])
	h.Sum(output[:0])
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample4: got %s, want %s", got, expected)
	}
}

func TestTupleHashNISTSample5(t *testing.T) {
	outputLength := 64
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
	}
	S := []byte("My Tuple App")
	output := make([]byte, outputLength)
	TupleHash256(tuples, S, output)
	expected := strings.Replace("14 7C 21 91 D5 ED 7E FD 98 DB D9 6D 7A B5 A1 16 92 57 6F 5F E2 A5 06 5F 3E 33 DE 6B BA 9F 3A A1 C4 E9 A0 68 A2 89 C6 1C 95 AA B3 0A EE 1E 41 0B 0B 60 7D E3 62 0E 24 A4 E3 BF 98 52 A1 D4 36 7E", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample5: got %s, want %s", got, expected)
	}

	h := NewTupleHash256(outputLength, S)
	for _, item := range tuples {
		h.Write(item)
	}
	copy(output, zero[:])
	h.Sum(output[:0])
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample5: got %s, want %s", got, expected)
	}
}

func TestTupleHashNISTSample6(t *testing.T) {
	outputLength := 64
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
		[]byte{0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28},
	}
	S := []byte("My Tuple App")
	output := make([]byte, outputLength)
	TupleHash256(tuples, S, output)
	expected := strings.Replace("45 00 0B E6 3F 9B 6B FD 89 F5 47 17 67 0F 69 A9 BC 76 35 91 A4 F0 5C 50 D6 88 91 A7 44 BC C6 E7 D6 D5 B5 E8 2C 01 8D A9 99 ED 35 B0 BB 49 C9 67 8E 52 6A BD 8E 85 C1 3E D2 54 02 1D B9 E7 90 CE", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample6: got %s, want %s", got, expected)
	}

	h := NewTupleHash256(outputLength, S)
	for _, item := range tuples {
		h.Write(item)
	}
	copy(output, zero[:])
	h.Sum(output[:0])
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashNISTSample6: got %s, want %s", got, expected)
	}
}

func TestTupleHashXOFNISTSample1(t *testing.T) {
	outputLength := 32
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
	}
	S := []byte{}
	r := TupleHashXOF128(tuples, S)
	output := make([]byte, outputLength)
	r.Read(output)
	expected := strings.Replace("2F 10 3C D7 C3 23 20 35 34 95 C6 8D E1 A8 12 92 45 C6 32 5F 6F 2A 3D 60 8D 92 17 9C 96 E6 84 88", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashXOFNISTSample1: got %s, want %s", got, expected)
	}

	h := NewTupleHashXOF128(S)
	for _, item := range tuples {
		h.Write(item)
	}
	copy(output, zero[:])
	h.Read(output)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashXOFNISTSample1: got %s, want %s", got, expected)
	}
}

func TestTupleHashXOFNISTSample2(t *testing.T) {
	outputLength := 32
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
	}
	S := []byte("My Tuple App")
	h := TupleHashXOF128(tuples, S)
	output := make([]byte, outputLength)
	h.Read(output)
	expected := strings.Replace("3F C8 AD 69 45 31 28 29 28 59 A1 8B 6C 67 D7 AD 85 F0 1B 32 81 5E 22 CE 83 9C 49 EC 37 4E 9B 9A", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashXOFNISTSample2: got %s, want %s", got, expected)
	}
}

func TestTupleHashXOFNISTSample3(t *testing.T) {
	outputLength := 32
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
		[]byte{0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28},
	}
	S := []byte("My Tuple App")
	h := TupleHashXOF128(tuples, S)
	output := make([]byte, outputLength)
	h.Read(output)
	expected := strings.Replace("90 0F E1 6C AD 09 8D 28 E7 4D 63 2E D8 52 F9 9D AA B7 F7 DF 4D 99 E7 75 65 78 85 B4 BF 76 D6 F8", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashXOFNISTSample3: got %s, want %s", got, expected)
	}
}

func TestTupleHashXOFNISTSample4(t *testing.T) {
	outputLength := 64
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
	}
	S := []byte{}
	h := TupleHashXOF256(tuples, S)
	output := make([]byte, outputLength)
	h.Read(output)
	expected := strings.Replace("03 DE D4 61 0E D6 45 0A 1E 3F 8B C4 49 51 D1 4F BC 38 4A B0 EF E5 7B 00 0D F6 B6 DF 5A AE 7C D5 68 E7 73 77 DA F1 3F 37 EC 75 CF 5F C5 98 B6 84 1D 51 DD 20 7C 99 1C D4 5D 21 0B A6 0A C5 2E B9", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashXOFNISTSample4: got %s, want %s", got, expected)
	}
}

func TestTupleHashXOFNISTSample5(t *testing.T) {
	outputLength := 64
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
	}
	S := []byte("My Tuple App")
	h := TupleHashXOF256(tuples, S)
	output := make([]byte, outputLength)
	h.Read(output)
	expected := strings.Replace("64 83 CB 3C 99 52 EB 20 E8 30 AF 47 85 85 1F C5 97 EE 3B F9 3B B7 60 2C 0E F6 A6 5D 74 1A EC A7 E6 3C 3B 12 89 81 AA 05 C6 D2 74 38 C7 9D 27 54 BB 1B 71 91 F1 25 D6 62 0F CA 12 CE 65 8B 24 42", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashXOFNISTSample5: got %s, want %s", got, expected)
	}
}

func TestTupleHashXOFNISTSample6(t *testing.T) {
	outputLength := 64
	tuples := [][]byte{
		[]byte{0x00, 0x01, 0x02},
		[]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
		[]byte{0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28},
	}
	S := []byte("My Tuple App")
	h := TupleHashXOF256(tuples, S)
	output := make([]byte, outputLength)
	h.Read(output)
	expected := strings.Replace("0C 59 B1 14 64 F2 33 6C 34 66 3E D5 1B 2B 95 0B EC 74 36 10 85 6F 36 C2 8D 1D 08 8D 8A 24 46 28 4D D0 98 30 A6 A1 78 DC 75 23 76 19 9F AE 93 5D 86 CF DE E5 91 3D 49 22 DF D3 69 B6 6A 53 C8 97", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestTupleHashXOFNISTSample6: got %s, want %s", got, expected)
	}
}
