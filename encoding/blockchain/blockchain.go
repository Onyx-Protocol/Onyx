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
var readerPool = sync.Pool{New: func() interface{} { return new(byteReader) }}

func getReader(r io.Reader) *byteReader {
	br := readerPool.Get().(*byteReader)
	br.reset(r)
	return br
}

var ErrRange = errors.New("value out of range")

func ReadVarint31(r io.Reader) (uint32, int, error) {
	br := getReader(r)
	defer readerPool.Put(br)
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
	br := getReader(r)
	defer readerPool.Put(br)
	val, err := binary.ReadUvarint(br)
	if err != nil {
		return 0, br.n, err
	}
	if val > math.MaxInt64 {
		return 0, br.n, ErrRange
	}
	return val, br.n, nil
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

// ReadVarstrList reads a varint31 length prefix followed by that many
// varstrs.
func ReadVarstrList(r io.Reader) ([][]byte, int, error) {
	nelts, n, err := ReadVarint31(r)
	if err != nil {
		return nil, n, err
	}
	if nelts == 0 {
		return nil, n, nil
	}
	result := make([][]byte, 0, nelts)
	for ; nelts > 0; nelts-- {
		s, n2, err := ReadVarstr31(r)
		n += n2
		if err != nil {
			return nil, n, err
		}
		result = append(result, s)
	}
	return result, n, nil
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

// ReadExtensibleString reads a varint31 length prefix and that many
// bytes from r. It then calls the given function to consume those
// bytes, returning any unconsumed suffix.
func ReadExtensibleString(r io.Reader, f func(io.Reader) error) ([]byte, int, error) {
	s, n, err := ReadVarstr31(r)
	if err != nil {
		return nil, n, err
	}
	sr := bytes.NewReader(s)
	err = f(sr)
	if err != nil {
		return nil, n, err
	}
	suffix, err := ioutil.ReadAll(sr)
	return suffix, n, err
}

// byteReader wraps io.Reader, satisfies io.ByteReader, keeps a
// count of the number of bytes read, and has sticky errors
type byteReader struct {
	n int
	r io.Reader
	e error
	b [1]byte
}

func (r *byteReader) reset(reader io.Reader) {
	*r = byteReader{n: 0, r: reader, e: nil}
}

func (r *byteReader) ReadByte() (byte, error) {
	if r.e != nil {
		return 0, r.e
	}
	n, err := r.r.Read(r.b[:])
	if n > 0 {
		// If there was an error, don't return it now, to prevent the
		// caller from ignoring the valid byte. Hold onto the error and
		// return it on the next call.
		// (See https://github.com/chain/chain/pull/1911#discussion_r80809872)
		r.e = err
		r.n++
		return r.b[0], nil
	}
	return 0, err
}
