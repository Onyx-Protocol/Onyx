package blockchain

import (
	"bytes"
	"io"
	"testing"
	"testing/iotest"
)

func TestByteReader(t *testing.T) {
	r := byteReader{r: reader{bytes.NewReader([]byte{1})}}
	c, err := r.ReadByte()
	if err != nil {
		t.Errorf("err = %v want nil", err)
	}
	if c != 1 {
		t.Errorf("c = %d want 1", c)
	}
	_, err = r.ReadByte()
	if err != io.EOF {
		t.Errorf("err = %v want %v", err, io.EOF)
	}
}

func TestDataErrByteReader(t *testing.T) {
	r := byteReader{r: iotest.DataErrReader(bytes.NewReader([]byte{1}))}
	c, err := r.ReadByte()
	if err != nil {
		t.Errorf("err = %v want nil", err)
	}
	if c != 1 {
		t.Errorf("c = %d want 1", c)
	}
	_, err = r.ReadByte()
	if err != io.EOF {
		t.Errorf("err = %v want %v", err, io.EOF)
	}
}

type reader struct{ io.Reader }
