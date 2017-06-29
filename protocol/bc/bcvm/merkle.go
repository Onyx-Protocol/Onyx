package bcvm

import (
	"math"

	"chain/crypto/sha3pool"
	"chain/protocol/bc"
)

var (
	leafPrefix     = []byte{0x00}
	interiorPrefix = []byte{0x01}
)

// MerkleRoot creates a merkle tree from a slice of transactions
// and returns the root hash of the tree.
func MerkleRoot(ids []bc.Hash) (root bc.Hash, err error) {
	switch {
	case len(ids) == 0:
		return bc.EmptyStringHash, nil

	case len(ids) == 1:
		h := sha3pool.Get256()
		defer sha3pool.Put256(h)

		h.Write(leafPrefix)
		ids[0].WriteTo(h)
		root.ReadFrom(h)
		return root, nil

	default:
		k := prevPowerOfTwo(len(ids))
		left, err := MerkleRoot(ids[:k])
		if err != nil {
			return root, err
		}

		right, err := MerkleRoot(ids[k:])
		if err != nil {
			return root, err
		}

		h := sha3pool.Get256()
		defer sha3pool.Put256(h)
		h.Write(interiorPrefix)
		left.WriteTo(h)
		right.WriteTo(h)
		root.ReadFrom(h)
		return root, nil
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
