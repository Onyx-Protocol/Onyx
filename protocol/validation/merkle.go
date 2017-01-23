package validation

import (
	"math"

	"chain/crypto/sha3pool"
	"chain/protocol/bc"
)

var (
	leafPrefix     = []byte{0x00}
	interiorPrefix = []byte{0x01}
)

// CalcMerkleRoot creates a merkle tree from a slice of transactions
// and returns the root hash of the tree.
func CalcMerkleRoot(transactions []*bc.Tx) (root bc.Hash, err error) {
	switch {
	case len(transactions) == 0:
		sha3pool.Sum256(root[:], nil)
		return root, nil

	case len(transactions) == 1:
		h := sha3pool.Get256()
		defer sha3pool.Put256(h)

		witHash, err := transactions[0].WitnessHash()
		if err != nil {
			return root, err
		}
		h.Write(leafPrefix)
		h.Write(witHash[:])
		h.Read(root[:])
		return root, nil

	default:
		k := prevPowerOfTwo(len(transactions))
		left, err := CalcMerkleRoot(transactions[:k])
		if err != nil {
			return root, err
		}
		right, err := CalcMerkleRoot(transactions[k:])
		if err != nil {
			return root, err
		}

		h := sha3pool.Get256()
		defer sha3pool.Put256(h)
		h.Write(interiorPrefix)
		h.Write(left[:])
		h.Write(right[:])
		h.Read(root[:])
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
