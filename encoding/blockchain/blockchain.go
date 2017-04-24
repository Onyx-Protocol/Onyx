// Package blockchain provides the tools for encoding
// data primitives in blockchain structures
package blockchain

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"sync"

	"chain/encoding/bufpool"
)

var bufPool = sync.Pool{New: func() interface{} { return new([9]byte) }}

var ErrRange = errors.New("value out of range")

// Reader wraps a buffer and provides utilities for decoding
// data primitives in blockchain structures. Its various read
// calls may return a slice of the underlying buffer.
type Reader struct {
	buf []byte
}

// NewReader constructs a new reader with the provided bytes. It
// does not create a copy of the bytes, so the caller is responsible
// for copying the bytes if necessary.
func NewReader(b []byte) *Reader {
	return &Reader{buf: b}
}

// Len returns the number of unread bytes.
func (r *Reader) Len() int {
	return len(r.buf)
}

// ReadByte reads and returns the next byte from the input.
//
// It implements the io.ByteReader interface.
func (r *Reader) ReadByte() (byte, error) {
	if len(r.buf) == 0 {
		return 0, io.EOF
	}

	b := r.buf[0]
	r.buf = r.buf[1:]
	return b, nil
}

// Read reads up to len(p) bytes into p. It implements
// the io.Reader interface.
func (r *Reader) Read(p []byte) (n int, err error) {
	n = copy(p, r.buf)
	r.buf = r.buf[n:]
	if len(r.buf) == 0 {
		err = io.EOF
	}
	return
}

func ReadVarint31(r *Reader) (uint32, error) {
	val, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, err
	}
	if val > math.MaxInt32 {
		return 0, ErrRange
	}
	return uint32(val), nil
}

func ReadVarint63(r *Reader) (uint64, error) {
	val, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, err
	}
	if val > math.MaxInt64 {
		return 0, ErrRange
	}
	return val, nil
}

func ReadVarstr31(r *Reader) ([]byte, error) {
	l, err := ReadVarint31(r)
	if err != nil {
		return nil, err
	}
	if l == 0 {
		return nil, nil
	}
	if int(l) > len(r.buf) {
		return nil, io.ErrUnexpectedEOF
	}
	str := r.buf[:l]
	r.buf = r.buf[l:]
	return str, nil
}

// ReadVarstrList reads a varint31 length prefix followed by
// that many varstrs.
func ReadVarstrList(r *Reader) (result [][]byte, err error) {
	nelts, err := ReadVarint31(r)
	if err != nil {
		return nil, err
	}
	if nelts == 0 {
		return nil, nil
	}

	for ; nelts > 0 && err == nil; nelts-- {
		var s []byte
		s, err = ReadVarstr31(r)
		result = append(result, s)
	}
	if len(result) < int(nelts) {
		err = io.ErrUnexpectedEOF
	}
	return result, err
}

// ReadExtensibleString reads a varint31 length prefix and that many
// bytes from r. It then calls the given function to consume those
// bytes, returning any unconsumed suffix.
func ReadExtensibleString(r *Reader, f func(*Reader) error) (suffix []byte, err error) {
	s, err := ReadVarstr31(r)
	if err != nil {
		return nil, err
	}

	sr := NewReader(s)
	err = f(sr)
	if err != nil {
		return nil, err
	}
	return sr.buf, nil
}

func WriteVarint31(w io.Writer, val uint64) (int, error) {
	if val > math.MaxInt32 {
		return 0, ErrRange
	}
	buf := bufPool.Get().(*[9]byte)
	n := binary.PutUvarint(buf[:], val)
	b, err := w.Write(buf[:n])
	bufPool.Put(buf)
	return b, err
}

func WriteVarint63(w io.Writer, val uint64) (int, error) {
	if val > math.MaxInt64 {
		return 0, ErrRange
	}
	buf := bufPool.Get().(*[9]byte)
	n := binary.PutUvarint(buf[:], val)
	b, err := w.Write(buf[:n])
	bufPool.Put(buf)
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

// WriteVarstrList writes a varint31 length prefix followed by the
// elements of l as varstrs.
func WriteVarstrList(w io.Writer, l [][]byte) (int, error) {
	n, err := WriteVarint31(w, uint64(len(l)))
	if err != nil {
		return n, err
	}
	for _, s := range l {
		n2, err := WriteVarstr31(w, s)
		n += n2
		if err != nil {
			return n, err
		}
	}
	return n, err
}

// WriteExtensibleString sends the output of the given function, plus
// the given suffix, to w, together with a varint31 length prefix.
func WriteExtensibleString(w io.Writer, suffix []byte, f func(io.Writer) error) (int, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)
	err := f(buf)
	if err != nil {
		return 0, err
	}
	if len(suffix) > 0 {
		_, err := buf.Write(suffix)
		if err != nil {
			return 0, err
		}
	}
	return WriteVarstr31(w, buf.Bytes())
}
