package ca

import (
	"crypto/subtle"
	"encoding/binary"
	"hash"

	"golang.org/x/crypto/sha3"

	"chain-stealth/crypto/sha3pool"
)

func hash256(input ...[]byte) (output [32]byte) {
	hash := sha3pool.Get256()
	for _, slice := range input {
		hash.Write(slice) // error is impossible
	}
	hash.Read(output[:])
	sha3pool.Put256(hash)
	return output
}

func hash512(input ...[]byte) (output [64]byte) {
	hash := sha3.New512()
	for _, slice := range input {
		hash.Write(slice) // error is impossible
	}
	hash.Sum(output[:0])
	return output
}

func hasher256(input ...[]byte) hash.Hash {
	hash := sha3.New256()
	for _, slice := range input {
		hash.Write(slice) // error is impossible
	}
	return hash
}

func hasher512(input ...[]byte) hash.Hash {
	hash := sha3.New512()
	for _, slice := range input {
		hash.Write(slice) // error is impossible
	}
	return hash
}

// Returns a hash instance ready to be Read() any number of times in any stride sizes necessary.
func shake256(input ...[]byte) sha3.ShakeHash {
	hash := sha3.NewShake256()
	for _, slice := range input {
		hash.Write(slice) // error is impossible
	}
	return hash
}

func uint256le(value uint64) (result [32]byte) {
	binary.LittleEndian.PutUint64(result[:8], value)
	return result
}

func uint64le(value uint64) (result []byte) {
	result = make([]byte, 8)
	binary.LittleEndian.PutUint64(result[:8], value)
	return result
}

func constTimeEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

func xor64(a []byte, b []byte) (r [8]byte) {
	// FIXME: make this more efficient by doing a single word XORing
	for i := 0; i < 8; i++ {
		r[i] = a[i] ^ b[i]
	}
	return r
}

func xor256(a []byte, b []byte) (r [32]byte) {
	// FIXME: make this more efficient by doing word-by-word XORing
	for i := 0; i < 32; i++ {
		r[i] = a[i] ^ b[i]
	}
	return r
}
