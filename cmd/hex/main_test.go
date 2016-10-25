package main

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
)

func TestStripSpace(t *testing.T) {
	in := "a b c"

	got, err := ioutil.ReadAll(&stripSpaceReader{strings.NewReader(in)})
	if err != nil {
		t.Fatal(err)
	}

	want := []byte("abc")
	if !bytes.Equal(got, want) {
		t.Errorf("stripSpaceReader(%q) = %q want %q", in, got, want)
	}
}

func TestEncode(t *testing.T) {
	in := "\xab\xcd"

	got, err := ioutil.ReadAll(&encodeReader{strings.NewReader(in)})
	if err != nil {
		t.Fatal(err)
	}

	want := []byte("abcd")
	if !bytes.Equal(got, want) {
		t.Errorf("encodeReader(%q) = %q want %q", in, got, want)
	}
}

func TestDecode(t *testing.T) {
	in := "abcd"

	got, err := ioutil.ReadAll(&decodeReader{r: strings.NewReader(in)})
	if err != nil {
		t.Fatal(err)
	}

	want := []byte{0xab, 0xcd}
	if !bytes.Equal(got, want) {
		t.Errorf("decodeReader(%q) = %q want %q", in, got, want)
	}
}
