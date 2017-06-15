package ca

import (
	"crypto/subtle"
	"encoding/binary"
)

func constTimeEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

func uint64le(value uint64) (result []byte) {
	result = make([]byte, 8)
	binary.LittleEndian.PutUint64(result[:8], value)
	return result
}

func xorSlices(a []byte, b []byte, out []byte) {
	n := len(out)
	for i := 0; i < n; i++ {
		out[i] = a[i] ^ b[i]
	}
	// TODO: check if this word-by-word implementation is faster than byte-by-byte one:
	// n := len(out) / 8
	// for i := 0; i < n; i++ {
	// 	x := binary.LittleEndian.Uint64(a)
	// 	y := binary.LittleEndian.Uint64(b)
	// 	x ^= y
	// 	binary.LittleEndian.PutUint64(out, x)
	// 	a = a[8:]
	// 	b = b[8:]
	// 	out = out[8:]
	// }
}
