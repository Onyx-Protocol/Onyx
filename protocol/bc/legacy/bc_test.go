package legacy

import (
	"bytes"
	"io"
	"testing"

	"chain/protocol/bc"
)

func serialize(t *testing.T, wt io.WriterTo) []byte {
	var b bytes.Buffer
	_, err := wt.WriteTo(&b)
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func mustDecodeHash(s string) (h bc.Hash) {
	err := h.UnmarshalText([]byte(s))
	if err != nil {
		panic(err)
	}
	return h
}
