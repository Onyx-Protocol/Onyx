package ca

import (
	"bytes"
	"reflect"
	"testing"
)

func TestEmptyPacket(t *testing.T) {
	ek := []byte("encryption key")

	pt := []byte{}
	got := EncryptPacket(ek, []byte{}, pt)
	want := EncryptedPacket{
		ct:    []byte{},
		nonce: [8]byte{0x04, 0x30, 0x72, 0x02, 0x1d, 0xf9, 0xe1, 0xcf},
		mac:   [24]byte{0x7c, 0x60, 0x04, 0x7b, 0xe4, 0xaa, 0x3b, 0x91, 0x00, 0x88, 0xa7, 0x8e, 0x66, 0xc4, 0xfa, 0x5d, 0xb0, 0xb1, 0x79, 0x0b, 0x50, 0x78, 0x41, 0xfa},
	}
	if !reflect.DeepEqual(got, &want) {
		t.Errorf("Got %v, want %v", got, want)
	}
	pt2, ok := got.Decrypt(ek)
	if !ok {
		t.Fatal("Failed to authenticate ciphertext")
	}
	if !bytes.Equal(pt, pt2) {
		t.Errorf("Got %x, want %x", pt2, pt)
	}
}

func TestShortPacket(t *testing.T) {
	ek := []byte("encryption key")

	pt := []byte{0x01}
	got := EncryptPacket(ek, []byte{}, pt)
	want := EncryptedPacket{
		ct:    []byte{0x5d},
		nonce: [8]byte{0x27, 0xcf, 0x50, 0x4c, 0x74, 0x66, 0x64, 0x87},
		mac:   [24]byte{0x72, 0x9d, 0xe0, 0x6d, 0x2a, 0xb1, 0x4c, 0xff, 0x10, 0x92, 0x08, 0xa1, 0x83, 0x1a, 0xfe, 0x86, 0x0a, 0x14, 0xd2, 0x99, 0x60, 0x9d, 0xe8, 0x1b},
	}
	if !reflect.DeepEqual(got, &want) {
		t.Errorf("Got %v, want %v", got, want)
	}
	pt2, ok := got.Decrypt(ek)
	if !ok {
		t.Fatal("Failed to authenticate ciphertext")
	}
	if !bytes.Equal(pt, pt2) {
		t.Errorf("Got %x, want %x", pt2, pt)
	}
}

func TestLongPacket(t *testing.T) {
	ek := []byte("encryption key")

	pt := make([]byte, 321)
	for i := 0; i < len(pt); i++ {
		pt[i] = byte(i % 256)
	}
	got := EncryptPacket(ek, []byte{}, pt)
	want := EncryptedPacket{
		ct:    fromHex("dbe2978388a2e009831ea37ad0b488060ad82b6a68994817dd9452c27cd6beaca4fbe01a351983e22396a27b245fa0ded0ede2f653e9ac14f1b8aeb2e2f0aa26fbef8b90d2f3abb152fde477a881f4f636f1273351c368011077429d71192f1d88e90d2bb7b1adaf0dc9707c52b3df2e0a936fbd48b69a169bf0c678e9413a498271b8ea6b9b487af085b29b9989890b48d04f2de95ba1c02eb33cd5b8025c526bf0622626c4428d426ba457d71e8517dc8a8d14b02d12a64c436dc7f022d93769651e82dc99df64f35b5266e411fb4930d527c3331a1f7490dff064d32974734351065b5ae2ef6fcb24c58b3e052e40387cd8b71758b5f01c21fac93f7ed7d4c44bcd77c31e7460124a9a5962b133ae9ebb5366b1dd74a307836e275769fa256677f0598e287a5e454c8d7ccef6e76bec2bf8e8417137234b94d4d3d3215f11ed"),
		nonce: [8]byte{0x5c, 0x67, 0x78, 0x1a, 0x04, 0xbd, 0x44, 0x0a},
		mac:   [24]byte{0xab, 0x9f, 0xca, 0xc3, 0x99, 0xb8, 0xea, 0x8c, 0x9f, 0x7d, 0xbc, 0x46, 0x78, 0xbb, 0x84, 0xfb, 0x11, 0xbd, 0x21, 0xd4, 0x38, 0x63, 0xb4, 0x77},
	}
	if !reflect.DeepEqual(got, &want) {
		t.Errorf("Got %v, want %v", got, want)
	}
	pt2, ok := got.Decrypt(ek)
	if !ok {
		t.Fatal("Failed to authenticate ciphertext")
	}
	if !bytes.Equal(pt, pt2) {
		t.Errorf("Got %x, want %x", pt2, pt)
	}
}
