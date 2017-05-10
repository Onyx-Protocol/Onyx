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
