// Package io provides I/O primitives supplemental
// to the standard library package io.
package io

import "io"

// ByteReader returns a ByteReader that makes one read on r
// for every call to ReadByte.
// If the underlying reader returns a byte and an error
// in the same call,
// ReadByte discards the error.
func ByteReader(r io.Reader) io.ByteReader {
	if br, ok := r.(io.ByteReader); ok {
		return br
	}
	return &byteReader{r: r}
}

type byteReader struct {
	r io.Reader
}

func (r *byteReader) ReadByte() (byte, error) {
	var b [1]byte
	n, err := r.r.Read(b[:])
	if n > 0 {
		err = nil // got one byte, so throw away err, oh well
	}
	return b[0], err
}
