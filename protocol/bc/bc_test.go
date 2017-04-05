package bc

import (
	"bytes"
	"encoding/hex"
	"io"
	"testing"
)

func serialize(t *testing.T, wt io.WriterTo) []byte {
	var b bytes.Buffer
	_, err := wt.WriteTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func mustDecodeHash(s string) (h Hash) {
	var b32 byte32
	if len(s) != hex.EncodedLen(len(b32)) {
		panic("wrong length hash")
	}
	_, err := hex.Decode(b32[:], []byte(s))
	if err != nil {
		panic(err)
	}
	h.FromByte32(b32)
	return h
}
