package sha3

// Test vectors from:
// - http://csrc.nist.gov/groups/ST/toolkit/documents/Examples/KMAC_samples.pdf
// - http://csrc.nist.gov/groups/ST/toolkit/documents/Examples/KMACXOF_samples.pdf

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestKMACNISTSample1(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := []byte{0x00, 0x01, 0x02, 0x03}
	outputLength := 32
	hash := NewKMAC128(key, outputLength, []byte{})
	hash.Write(data)
	output := make([]byte, outputLength)
	hash.Sum(output[:0])
	expected := strings.Replace("E5 78 0B 0D 3E A6 F7 D3 A4 29 C5 70 6A A4 3A 00 FA DB D7 D4 96 28 83 9E 31 87 24 3F 45 6E E1 4E", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACNISTSample1: got %s, want %s", got, expected)
	}
}

func TestKMACNISTSample2(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := []byte{0x00, 0x01, 0x02, 0x03}
	outputLength := 32
	hash := NewKMAC128(key, outputLength, []byte("My Tagged Application"))
	hash.Write(data)
	output := make([]byte, outputLength)
	hash.Sum(output[:0])
	expected := strings.Replace("3B 1F BA 96 3C D8 B0 B5 9E 8C 1A 6D 71 88 8B 71 43 65 1A F8 BA 0A 70 70 C0 97 9E 28 11 32 4A A5", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACNISTSample2: got %s, want %s", got, expected)
	}
}

func TestKMACNISTSample3(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := make([]byte, 1600/8) // 1600 bits: "00 01 02 03 .. C4 C5 C6 C7"
	for i := byte(0); i <= 0xc7; i++ {
		data[i] = i
	}
	outputLength := 32
	hash := NewKMAC128(key, outputLength, []byte("My Tagged Application"))
	hash.Write(data)
	output := make([]byte, outputLength)
	hash.Sum(output[:0])
	expected := strings.Replace("1F 5B 4E 6C CA 02 20 9E 0D CB 5C A6 35 B8 9A 15 E2 71 EC C7 60 07 1D FD 80 5F AA 38 F9 72 92 30", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACNISTSample3: got %s, want %s", got, expected)
	}
}

func TestKMACNISTSample4(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := []byte{0x00, 0x01, 0x02, 0x03}
	outputLength := 64
	hash := NewKMAC256(key, outputLength, []byte("My Tagged Application"))
	hash.Write(data)
	output := make([]byte, outputLength)
	hash.Sum(output[:0])
	expected := strings.Replace("20 C5 70 C3 13 46 F7 03 C9 AC 36 C6 1C 03 CB 64 C3 97 0D 0C FC 78 7E 9B 79 59 9D 27 3A 68 D2 F7 F6 9D 4C C3 DE 9D 10 4A 35 16 89 F2 7C F6 F5 95 1F 01 03 F3 3F 4F 24 87 10 24 D9 C2 77 73 A8 DD", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACNISTSample4: got %s, want %s", got, expected)
	}
}

func TestKMACNISTSample5(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := make([]byte, 1600/8) // 1600 bits: "00 01 02 03 .. C4 C5 C6 C7"
	for i := byte(0); i <= 0xc7; i++ {
		data[i] = i
	}
	outputLength := 64
	hash := NewKMAC256(key, outputLength, []byte{})
	hash.Write(data)
	output := make([]byte, outputLength)
	hash.Sum(output[:0])
	expected := strings.Replace("75 35 8C F3 9E 41 49 4E 94 97 07 92 7C EE 0A F2 0A 3F F5 53 90 4C 86 B0 8F 21 CC 41 4B CF D6 91 58 9D 27 CF 5E 15 36 9C BB FF 8B 9A 4C 2E B1 78 00 85 5D 02 35 FF 63 5D A8 25 33 EC 6B 75 9B 69", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACNISTSample5: got %s, want %s", got, expected)
	}
}

func TestKMACNISTSample6(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := make([]byte, 1600/8) // 1600 bits: "00 01 02 03 .. C4 C5 C6 C7"
	for i := byte(0); i <= 0xc7; i++ {
		data[i] = i
	}
	outputLength := 64
	hash := NewKMAC256(key, outputLength, []byte("My Tagged Application"))
	hash.Write(data)
	output := make([]byte, outputLength)
	hash.Sum(output[:0])
	expected := strings.Replace("B5 86 18 F7 1F 92 E1 D5 6C 1B 8C 55 DD D7 CD 18 8B 97 B4 CA 4D 99 83 1E B2 69 9A 83 7D A2 E4 D9 70 FB AC FD E5 00 33 AE A5 85 F1 A2 70 85 10 C3 2D 07 88 08 01 BD 18 28 98 FE 47 68 76 FC 89 65", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACNISTSample6: got %s, want %s", got, expected)
	}
}

func TestKMACXOFNISTSample1(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := []byte{0x00, 0x01, 0x02, 0x03}
	outputLength := 32
	shake := NewKMACXOF128(key, []byte{})
	shake.Write(data)
	output := make([]byte, outputLength)
	shake.Read(output)
	expected := strings.Replace("CD 83 74 0B BD 92 CC C8 CF 03 2B 14 81 A0 F4 46 0E 7C A9 DD 12 B0 8A 0C 40 31 17 8B AC D6 EC 35", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACXOFNISTSample1: got %s, want %s", got, expected)
	}
}

func TestKMACXOFNISTSample2(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := []byte{0x00, 0x01, 0x02, 0x03}
	outputLength := 32
	shake := NewKMACXOF128(key, []byte("My Tagged Application"))
	shake.Write(data)
	output := make([]byte, outputLength)
	shake.Read(output)
	expected := strings.Replace("31 A4 45 27 B4 ED 9F 5C 61 01 D1 1D E6 D2 6F 06 20 AA 5C 34 1D EF 41 29 96 57 FE 9D F1 A3 B1 6C", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACXOFNISTSample2: got %s, want %s", got, expected)
	}
}

func TestKMACXOFNISTSample3(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := make([]byte, 1600/8) // 1600 bits: "00 01 02 03 .. C4 C5 C6 C7"
	for i := byte(0); i <= 0xc7; i++ {
		data[i] = i
	}
	outputLength := 32
	shake := NewKMACXOF128(key, []byte("My Tagged Application"))
	shake.Write(data)
	output := make([]byte, outputLength)
	shake.Read(output)
	expected := strings.Replace("47 02 6C 7C D7 93 08 4A A0 28 3C 25 3E F6 58 49 0C 0D B6 14 38 B8 32 6F E9 BD DF 28 1B 83 AE 0F", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACXOFNISTSample3: got %s, want %s", got, expected)
	}
}

func TestKMACXOFNISTSample4(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := []byte{0x00, 0x01, 0x02, 0x03}
	outputLength := 64
	shake := NewKMACXOF256(key, []byte("My Tagged Application"))
	shake.Write(data)
	output := make([]byte, outputLength)
	shake.Read(output)
	expected := strings.Replace("17 55 13 3F 15 34 75 2A AD 07 48 F2 C7 06 FB 5C 78 45 12 CA B8 35 CD 15 67 6B 16 C0 C6 64 7F A9 6F AA 7A F6 34 A0 BF 8F F6 DF 39 37 4F A0 0F AD 9A 39 E3 22 A7 C9 20 65 A6 4E B1 FB 08 01 EB 2B", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACXOFNISTSample4: got %s, want %s", got, expected)
	}
}

func TestKMACXOFNISTSample5(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := make([]byte, 1600/8) // 1600 bits: "00 01 02 03 .. C4 C5 C6 C7"
	for i := byte(0); i <= 0xc7; i++ {
		data[i] = i
	}
	outputLength := 64
	shake := NewKMACXOF256(key, []byte{})
	shake.Write(data)
	output := make([]byte, outputLength)
	shake.Read(output)
	expected := strings.Replace("FF 7B 17 1F 1E 8A 2B 24 68 3E ED 37 83 0E E7 97 53 8B A8 DC 56 3F 6D A1 E6 67 39 1A 75 ED C0 2C A6 33 07 9F 81 CE 12 A2 5F 45 61 5E C8 99 72 03 1D 18 33 73 31 D2 4C EB 8F 8C A8 E6 A1 9F D9 8B", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACXOFNISTSample5: got %s, want %s", got, expected)
	}
}

func TestKMACXOFNISTSample6(t *testing.T) {
	key, _ := hex.DecodeString(strings.Replace("40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F 50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F", " ", "", -1))
	data := make([]byte, 1600/8) // 1600 bits: "00 01 02 03 .. C4 C5 C6 C7"
	for i := byte(0); i <= 0xc7; i++ {
		data[i] = i
	}
	outputLength := 64
	shake := NewKMACXOF256(key, []byte("My Tagged Application"))
	shake.Write(data)
	output := make([]byte, outputLength)
	shake.Read(output)
	expected := strings.Replace("D5 BE 73 1C 95 4E D7 73 28 46 BB 59 DB E3 A8 E3 0F 83 E7 7A 4B FF 44 59 F2 F1 C2 B4 EC EB B8 CE 67 BA 01 C6 2E 8A B8 57 8D 2D 49 9B D1 BB 27 67 68 78 11 90 02 0A 30 6A 97 DE 28 1D CC 30 30 5D", " ", "", -1)
	if got := strings.ToUpper(hex.EncodeToString(output)); got != expected {
		t.Errorf("TestKMACXOFNISTSample5: got %s, want %s", got, expected)
	}
}
