package io

import (
	"bytes"
	"io"
	"testing"
	"testing/iotest"
)

func TestByteReader(t *testing.T) {
	r := ByteReader(reader{bytes.NewReader([]byte{1})})
	c, err := r.ReadByte()
	if err != nil {
		t.Errorf("err = %v want nil", err)
	}
	if c != 1 {
		t.Errorf("c = %d want 1", c)
	}
	c, err = r.ReadByte()
	if err != io.EOF {
		t.Errorf("err = %v want %v", err, io.EOF)
	}
}

func TestDataErrByteReader(t *testing.T) {
	r := ByteReader(iotest.DataErrReader(bytes.NewReader([]byte{1})))
	c, err := r.ReadByte()
	if err != nil {
		t.Errorf("err = %v want nil", err)
	}
	if c != 1 {
		t.Errorf("c = %d want 1", c)
	}
	c, err = r.ReadByte()
	if err != io.EOF {
		t.Errorf("err = %v want %v", err, io.EOF)
	}
}

func TestIdentityByteReader(t *testing.T) {
	br := bytes.NewReader(nil)
	got := ByteReader(br)
	if got != br {
		t.Errorf("byte reader = %#v want %#v", got, br)
	}
}

type reader struct{ io.Reader }
