// Package blockchain provides the tools for encoding
// data primitives in blockchain structures
package blockchain

import (
	"encoding/binary"
	"io"

	chainio "chain/io"
)

// endianness is the default endian encoding (little or big)
var endianness = binary.LittleEndian

// WriteUvarint writes a variable-length unsigned int
func WriteUvarint(w io.Writer, x uint64) error {
	var buf [9]byte
	n := binary.PutUvarint(buf[:], x)
	_, err := w.Write(buf[0:n])
	return err
}

// ReadUvarint reads a variable-length unsigned int
func ReadUvarint(r io.Reader) (uint64, error) {
	return binary.ReadUvarint(chainio.ByteReader(r))
}

// WriteUint32 writes a fixed-length uint32
func WriteUint32(w io.Writer, x uint32) (int, error) {
	var buf [4]byte
	endianness.PutUint32(buf[:], x)
	return w.Write(buf[:])
}

// ReadUint32 reads a fixed-length uint32
func ReadUint32(r io.Reader) (uint32, error) {
	var buf [4]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return 0, err
	}
	return endianness.Uint32(buf[:]), nil
}

// WriteUint64 writes a fixed-length uint64
func WriteUint64(w io.Writer, x uint64) error {
	var buf [8]byte
	endianness.PutUint64(buf[:], x)
	_, err := w.Write(buf[:])
	return err
}

// ReadUint64 reads a fixed-length uint64
func ReadUint64(r io.Reader) (uint64, error) {
	var buf [8]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return 0, err
	}
	return endianness.Uint64(buf[:]), nil
}

// WriteBytes writes the length of the byte slice
// followed by the bytes.
func WriteBytes(w io.Writer, data []byte) error {
	err := WriteUvarint(w, uint64(len(data)))
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ReadBytes reads the length of the byte slice,
// then reads the bytes.
func ReadBytes(r io.Reader, b *[]byte) error {
	n, err := ReadUvarint(r)
	if n < 1 || err != nil {
		return err // can be successful read of 0
	}
	*b = make([]byte, n)
	_, err = io.ReadFull(r, *b)
	return err
}
