// Package blockchain provides the tools for encoding
// data primitives in blockchain structures
package blockchain

import (
	"encoding/binary"
	"fmt"
	"io"

	chainio "chain/io"
)

// WriteUvarint writes a variable-length unsigned int
func WriteUvarint(w io.Writer, x uint64) (int, error) {
	var buf [9]byte
	n := binary.PutUvarint(buf[:], x)
	return w.Write(buf[0:n])
}

// ReadUvarint reads a variable-length unsigned int
func ReadUvarint(r io.Reader) (uint64, error) {
	return binary.ReadUvarint(chainio.ByteReader(r))
}

// WriteBytes writes the length of the byte slice
// followed by the bytes.
func WriteBytes(w io.Writer, data []byte) error {
	_, err := WriteUvarint(w, uint64(len(data)))
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ReadBytes reads the length of the byte slice, then reads the bytes.
// If the byte slice exceeds the provided max, ReadBytes returns an error.
// Since max errors are not I/O related, they will not be captured by an
// io.Reader with sticky errors and must be handled by the caller.
func ReadBytes(r io.Reader, max uint64) ([]byte, error) {
	n, err := ReadUvarint(r)
	if n < 1 || err != nil {
		return nil, err // can be successful read of 0
	}
	if n > max {
		return nil, fmt.Errorf("cannot read %d bytes; max is %d", n, max)
	}
	b := make([]byte, n)
	_, err = io.ReadFull(r, b)
	return b, err
}
