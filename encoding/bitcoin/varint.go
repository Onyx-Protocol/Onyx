// Package bitcoin encodes and decodes numbers and strings
// using the Bitcoin varint format.
package bitcoin

import (
	"encoding/binary"
	"io"
	"math"
)

// ReadVarint reads a variable length integer in the Bitcoin varint format.
func ReadVarint(r io.Reader) (uint64, error) {
	var b [8]byte
	_, err := io.ReadFull(r, b[0:1])
	if err != nil {
		return 0, err
	}

	var v uint64
	switch b[0] {
	case 0xff:
		_, err = io.ReadFull(r, b[:])
		v = binary.LittleEndian.Uint64(b[:])
	case 0xfe:
		_, err = io.ReadFull(r, b[0:4])
		v = uint64(binary.LittleEndian.Uint32(b[:]))
	case 0xfd:
		_, err = io.ReadFull(r, b[0:2])
		v = uint64(binary.LittleEndian.Uint16(b[:]))
	default:
		v = uint64(b[0])
	}
	if err != nil {
		return 0, err
	}
	return v, nil
}

// WriteVarint serializes v to w in the Bitcoin varint format.
// It returns the number of bytes written.
func WriteVarint(w io.Writer, v uint64) (int, error) {
	var buf [9]byte
	switch {
	case v < 0xfd:
		return w.Write([]byte{uint8(v)})
	case v <= math.MaxUint16:
		buf[0] = 0xfd
		binary.LittleEndian.PutUint16(buf[1:], uint16(v))
		return w.Write(buf[:3])
	case v <= math.MaxUint32:
		buf[0] = 0xfe
		binary.LittleEndian.PutUint32(buf[1:], uint32(v))
		return w.Write(buf[:5])
	default: // v > math.MaxUint32
		buf[0] = 0xff
		binary.LittleEndian.PutUint64(buf[1:], v)
		return w.Write(buf[:9])
	}
}
