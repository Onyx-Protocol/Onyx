// Package blockchain provides the tools for encoding
// data primitives in blockchain structures
package blockchain

import (
	"encoding/binary"
	"io"

	"chain/errors"
	chainio "chain/io"
)

// endianness is the default endian encoding (little or big)
var endianness = binary.LittleEndian

// WriteUvarint writes a variable-length unsigned int
func WriteUvarint(w *errors.Writer, x uint64) {
	var buf [9]byte
	n := binary.PutUvarint(buf[:], x)
	w.Write(buf[0:n])
}

// ReadUvarint reads a variable-length unsigned int
func ReadUvarint(r *errors.Reader) uint64 {
	n, err := binary.ReadUvarint(chainio.ByteReader(r))
	if err != nil {
		r.Err = err
	}
	return n
}

// WriteUint32 writes a fixed-length uint32
func WriteUint32(w *errors.Writer, x uint32) {
	var buf [4]byte
	endianness.PutUint32(buf[:], x)
	w.Write(buf[:])
}

// ReadUint32 reads a fixed-length uint32
func ReadUint32(r *errors.Reader) uint32 {
	var buf [4]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		r.Err = err
	}
	return endianness.Uint32(buf[:])
}

// WriteUint64 writes a fixed-length uint64
func WriteUint64(w *errors.Writer, x uint64) {
	var buf [8]byte
	endianness.PutUint64(buf[:], x)
	w.Write(buf[:])
}

// ReadUint64 reads a fixed-length uint64
func ReadUint64(r *errors.Reader) uint64 {
	var buf [8]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		r.Err = err
	}
	return endianness.Uint64(buf[:])
}

// WriteBytes writes the length of the byte slice
// followed by the bytes.
func WriteBytes(w *errors.Writer, data []byte) {
	WriteUvarint(w, uint64(len(data)))
	w.Write(data)
}

// ReadBytes reads the length of the byte slice,
// then reads the bytes.
func ReadBytes(r *errors.Reader, b *[]byte) {
	n := ReadUvarint(r)
	if n < 1 {
		return
	}
	*b = make([]byte, n)
	io.ReadFull(r, *b)
}
