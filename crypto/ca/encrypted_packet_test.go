package ca

import (
	"bytes"
	"testing"
)

func TestEmptyPacket(t *testing.T) {
	ek := []byte("encryption key")

	pt := []byte{}
	ep := make([]byte, len(pt)+32)
	EncryptPacket(ek, []byte{}, pt, ep)

	want := fromHex("043072021df9e1cf7c60047be4aa3b910088a78e66c4fa5db0b1790b507841fa")
	got := ep
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}
	pt2 := make([]byte, len(pt))
	ok := DecryptPacket(ek, ep, pt2)
	if !ok {
		t.Fatal("Failed to authenticate ciphertext")
	}
	want = pt
	got = pt2
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}
}

func TestShortPacket(t *testing.T) {
	ek := []byte("encryption key")

	pt := []byte{0x01}
	ep := make([]byte, len(pt)+32)
	EncryptPacket(ek, []byte{}, pt, ep)

	want := fromHex("5d27cf504c74666487729de06d2ab14cff109208a1831afe860a14d299609de81b")
	got := ep
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}
	pt2 := make([]byte, len(pt))
	ok := DecryptPacket(ek, ep, pt2)
	if !ok {
		t.Fatal("Failed to authenticate ciphertext")
	}
	want = pt
	got = pt2
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}
}

func TestLongPacket(t *testing.T) {
	ek := []byte("encryption key")

	pt := make([]byte, 321)
	for i := 0; i < len(pt); i++ {
		pt[i] = byte(i % 256)
	}
	ep := make([]byte, len(pt)+32)
	EncryptPacket(ek, []byte{}, pt, ep)

	want := fromHex("dbe2978388a2e009831ea37ad0b488060ad82b6a68994817dd9452c27cd6beaca4fbe01a351983e22396a27b245fa0ded0ede2f653e9ac14f1b8aeb2e2f0aa26fbef8b90d2f3abb152fde477a881f4f636f1273351c368011077429d71192f1d88e90d2bb7b1adaf0dc9707c52b3df2e0a936fbd48b69a169bf0c678e9413a498271b8ea6b9b487af085b29b9989890b48d04f2de95ba1c02eb33cd5b8025c526bf0622626c4428d426ba457d71e8517dc8a8d14b02d12a64c436dc7f022d93769651e82dc99df64f35b5266e411fb4930d527c3331a1f7490dff064d32974734351065b5ae2ef6fcb24c58b3e052e40387cd8b71758b5f01c21fac93f7ed7d4c44bcd77c31e7460124a9a5962b133ae9ebb5366b1dd74a307836e275769fa256677f0598e287a5e454c8d7ccef6e76bec2bf8e8417137234b94d4d3d3215f11ed5c67781a04bd440aab9fcac399b8ea8c9f7dbc4678bb84fb11bd21d43863b477")
	got := ep
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}
	pt2 := make([]byte, len(pt))
	ok := DecryptPacket(ek, ep, pt2)
	if !ok {
		t.Fatal("Failed to authenticate ciphertext")
	}
	want = pt
	got = pt2
	if !bytes.Equal(got, want) {
		t.Errorf("Got %x, want %x", got, want)
	}
}
