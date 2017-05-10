package ca

import (
	"chain/crypto/ed25519/ecmath"
	"chain/crypto/sha3pool"

	"golang.org/x/crypto/sha3"
)

func sha3_256(input ...[]byte) (output [32]byte) {
	hash := sha3pool.Get256()
	for _, slice := range input {
		hash.Write(slice) // error is impossible
	}
	hash.Read(output[:])
	sha3pool.Put256(hash)
	return output
}

func hasher256(input ...[]byte) sha3.ShakeHash {
	// Hash256(x) = SHAKE128("ChainCA-256" || x, 32)
	hash := sha3.NewShake128()
	hash.Write([]byte("ChainCA-256"))
	for _, slice := range input {
		hash.Write(slice) // error is impossible
	}
	return hash
}

func hash256(input ...[]byte) (output [32]byte) {
	h := hasher256(input...)
	h.Read(output[:])
	return output
}

func streamHash(input ...[]byte) sha3.ShakeHash {
	// StreamHash(x, n) = SHAKE128("ChainCA-stream" || x, n)
	hash := sha3.NewShake128()
	hash.Write([]byte("ChainCA-stream"))
	for _, slice := range input {
		hash.Write(slice) // error is impossible
	}
	return hash
}

func scalarHasher(input ...[]byte) sha3.ShakeHash {
	// SHAKE128("ChainCA-scalar" || x, 64) mod L
	hash := sha3.NewShake128()
	hash.Write([]byte("ChainCA-scalar"))
	for _, slice := range input {
		hash.Write(slice) // error is impossible
	}
	return hash
}

func scalarHasherFinalize(h sha3.ShakeHash) (s ecmath.Scalar) {
	var buf [64]byte
	h.Read(buf[:])
	s.Reduce(&buf)
	return s
}

func scalarHash(input ...[]byte) ecmath.Scalar {
	return scalarHasherFinalize(scalarHasher(input...))
}
