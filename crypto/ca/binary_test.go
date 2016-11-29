package ca

import (
	"bytes"
	"testing"
)

func TestSHAKE256(t *testing.T) {
	var want []byte
	var got []byte

	want = fromHex("46b9dd2b0ba88d13233b3feb743eeb243fcd52ea62b81b82b50c27646ed5762f")
	buf32 := [32]byte{}
	got = buf32[:]
	shake256([]byte{}).Read(got)
	if !bytes.Equal(got, want) {
		t.Errorf("SHAKE256(32, '') is not correct:\ngot:  %x\nwant: %x", got, want)
	}

	want = fromHex("46b9dd2b0ba88d13233b3feb743eeb243fcd52ea62b81b82b50c27646ed5762fd75dc4ddd8c0f200cb05019d67b592f6fc821c49479ab48640292eacb3b7c4be")
	buf64 := [64]byte{}
	got = buf64[:]
	shake256([]byte{}).Read(got)
	if !bytes.Equal(got, want) {
		t.Errorf("SHAKE256(64, '') is not correct:\ngot:  %x\nwant: %x", got, want)
	}
}
