package bc

import (
	"bytes"
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

func decodeHash256(id string) (hash [32]byte) {
	err := DecodeHash256(id, &hash)
	if err != nil {
		panic(err)
	}
	return hash
}
