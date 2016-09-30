// Package blockchain provides the tools for encoding
// data primitives in blockchain structures
package blockchain

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

var ErrRange = errors.New("value out of range")

func ReadVarint31(r io.Reader) (uint32, int, error) {
	br := &byteReader{r: r}
	val, err := binary.ReadUvarint(br)
	if err != nil {
		return 0, br.n, err
	}
	if val > math.MaxInt32 {
		return 0, br.n, ErrRange
	}
	return uint32(val), br.n, nil
}

func ReadVarint63(r io.Reader) (uint64, int, error) {
	br := &byteReader{r: r}
	val, err := binary.ReadUvarint(br)
	if err != nil {
		return 0, br.n, err
	}
	if val > math.MaxInt64 {
		return 0, br.n, ErrRange
	}
	return uint64(val), br.n, nil
}

func ReadVarstr31(r io.Reader) ([]byte, int, error) {
	len, n, err := ReadVarint31(r)
	if err != nil {
		return nil, n, err
	}
	if len == 0 {
		return nil, n, nil
	}
	buf := make([]byte, len)
	n2, err := io.ReadFull(r, buf)
	return buf, n + n2, err
}

func WriteVarint31(w io.Writer, val uint64) (int, error) {
	if val > math.MaxInt32 {
		return 0, ErrRange
	}
	var buf [9]byte
	n := binary.PutUvarint(buf[:], val)
	b, err := w.Write(buf[:n])
	return b, err
}

func WriteVarint63(w io.Writer, val uint64) (int, error) {
	if val > math.MaxInt64 {
		return 0, ErrRange
	}
	var buf [9]byte
	n := binary.PutUvarint(buf[:], val)
	b, err := w.Write(buf[:n])
	return b, err
}

func WriteVarstr31(w io.Writer, str []byte) (int, error) {
	n, err := WriteVarint31(w, uint64(len(str)))
	if err != nil {
		return n, err
	}
	n2, err := w.Write(str)
	return n + n2, err
}

// byteReader wraps io.Reader, satisfies io.ByteReader, keeps a
// count of the number of bytes read, and has sticky errors
type byteReader struct {
	n int
	r io.Reader
	e error
}

func (r *byteReader) ReadByte() (byte, error) {
	if r.e != nil {
		return 0, r.e
	}
	var b [1]byte
	n, err := r.r.Read(b[:])
	if n > 0 {
		// If there was an error, don't return it now, to prevent the
		// caller from ignoring the valid byte. Hold onto the error and
		// return it on the next call.
		// (See https://github.com/chain/chain/pull/1911#discussion_r80809872)
		r.e = err
		r.n++
		return b[0], nil
	}
	return 0, err
}
