// Package blockchain provides the tools for encoding
// data primitives in blockchain structures
package blockchain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"math"
	"sync"

	"chain/encoding/bufpool"
)

var bufPool = sync.Pool{New: func() interface{} { return new([9]byte) }}

var ErrRange = errors.New("value out of range")

type Reader interface {
	io.Reader
	io.ByteReader
}

func ReadVarint31(r io.ByteReader) (uint32, error) {
	val, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, err
	}
	if val > math.MaxInt32 {
		return 0, ErrRange
	}
	return uint32(val), nil
}

func ReadVarint63(r io.ByteReader) (uint64, error) {
	val, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, err
	}
	if val > math.MaxInt64 {
		return 0, ErrRange
	}
	return val, nil
}

func ReadVarstr31(r Reader) ([]byte, error) {
	l, err := ReadVarint31(r)
	if err != nil {
		return nil, err
	}
	if l == 0 {
		return nil, nil
	}
	str := make([]byte, l)
	_, err = io.ReadFull(r, str)
	return str, err
}

// ReadVarstrList reads a varint31 length prefix followed by
// that many varstrs.
func ReadVarstrList(r Reader) (result [][]byte, err error) {
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
func ReadExtensibleString(r Reader, f func(Reader) error) (suffix []byte, err error) {
	s, err := ReadVarstr31(r)
	if err != nil {
		return nil, err
	}
	sr := bytes.NewReader(s)
	err = f(sr)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(sr)
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
