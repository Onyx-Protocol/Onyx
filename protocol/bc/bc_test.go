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
