package ca

import (
	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519/ecmath"
	"chain/crypto/sha3pool"
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

// hasher256 takes a customization string and an optional tuple.
// Customization string must be fully specified: e.g. `ChainCA.RS.e`.
func hasher256(s string, tuple ...[]byte) sha3.TupleHash {
	// Hash256(F,X) = TupleHash128(X, L=256, S="ChainCA." || F)
	hash := sha3.NewTupleHash128(32, []byte(s))
	for _, item := range tuple {
		hash.WriteItem(item)
	}
	return hash
}

// hash256 takes a customization string and a tuple.
// Customization string must be fully specified: e.g. `ChainCA.RS.e`.
func hash256(s string, tuple ...[]byte) (output [32]byte) {
	h := hasher256(s, tuple...)
	h.Sum(output[:0])
	return output
}

// streamHash takes a customization string with an optional tuple,
// and returns an instance of an extensible output function (XOF).
// Customization string must be fully specified: e.g. `ChainCA.BRS.Overlay`.
func streamHash(s string, tuple ...[]byte) sha3.TupleHashXOF {
	// StreamHash(F, X, n) = TupleHashXOF128(X, L=n·8, S="ChainCA." || F)
	hash := sha3.NewTupleHashXOF128([]byte(s))
	for _, item := range tuple {
		hash.WriteItem(item) // error is impossible
	}
	return hash
}

func scalarHasher(s string, tuple ...[]byte) sha3.TupleHash {
	// ScalarHash(F,X) = TupleHash128(X, L=512, S="ChainCA." || F)
	hash := sha3.NewTupleHash128(64, []byte(s))
	for _, item := range tuple {
		hash.WriteItem(item)
	}
	return hash
}

func scalarHasherFinalize(h sha3.TupleHash) (s ecmath.Scalar) {
	var buf [64]byte
	h.Sum(buf[:0])
	s.Reduce(&buf)
	return s
}

func scalarHash(s string, tuple ...[]byte) ecmath.Scalar {
	return scalarHasherFinalize(scalarHasher(s, tuple...))
}

func pointHash(s string, tuple ...[]byte) ecmath.Point {
	var result ecmath.Point
	counter := byte(0)
	for {
		h := hasher256(s)
		h.WriteItem([]byte{counter})
		for _, item := range tuple {
			h.WriteItem(item)
		}
		var hout [32]byte
		h.Sum(hout[:0])
		_, ok := result.Decode(hout)
		if ok {
			cofactor := ecmath.Scalar{8}
			result.ScMul(&result, &cofactor)
			return result
		}
		counter++
	}
}
