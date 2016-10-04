package validation

import (
	"math"

	"golang.org/x/crypto/sha3"

	"chain/protocol/bc"
)

var (
	leafPrefix     = []byte{0x00}
	interiorPrefix = []byte{0x01}
)

// CalcMerkleRoot creates a merkle tree from a slice of transactions
// and returns the root hash of the tree.
func CalcMerkleRoot(transactions []*bc.Tx) bc.Hash {
	switch {
	case len(transactions) == 0:
		return sha3.Sum256(nil)

	case len(transactions) == 1:
		witHash := transactions[0].WitnessHash()
		return sha3.Sum256(append(leafPrefix, witHash[:]...))

	default:
		k := prevPowerOfTwo(len(transactions))
		left := CalcMerkleRoot(transactions[:k])
		right := CalcMerkleRoot(transactions[k:])
		return sha3.Sum256(append(append(interiorPrefix, left[:]...), right[:]...))
	}
}

// prevPowerOfTwo returns the largest power of two that is smaller than a given number.
// In other words, for some input n, the prevPowerOfTwo k is a power of two such that
// k < n <= 2k. This is a helper function used during the calculation of a merkle tree.
func prevPowerOfTwo(n int) int {
	// If the number is a power of two, divide it by 2 and return.
	if n&(n-1) == 0 {
		return n / 2
	}

	// Otherwise, find the previous PoT.
	exponent := uint(math.Log2(float64(n)))
	return 1 << exponent // 2^exponent
}
