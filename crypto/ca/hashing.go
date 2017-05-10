package ca

import "chain/crypto/sha3pool"

func hash256(input ...[]byte) (output [32]byte) {
	hash := sha3pool.Get256()
	hash.Write([]byte("ChainCA-256"))
	for _, slice := range input {
		hash.Write(slice) // error is impossible
	}
	hash.Read(output[:])
	sha3pool.Put256(hash)
	return output
}

// #### StreamHash

// `StreamHash` is a secure extendable-output hash function that takes a variable-length binary string `x` as input
// and outputs a variable-length hash string depending on a number of bytes (`n`) requested.

//     StreamHash(x, n) = SHAKE128("ChainCA-stream" || x, n)

// #### ScalarHash

// `ScalarHash` is a secure hash function that takes a variable-length binary string `x` as input and outputs a [scalar](#scalar):

// 1. For the input string `x` compute a 512-bit hash `h`:

//         h = SHAKE128("ChainCA-scalar" || x, 64)

// 2. Interpret `h` as a little-endian integer and reduce modulo subgroup [order](#elliptic-curve) `L`:

//         s = h mod L

// 3. Return the resulting scalar `s`.
