package ecmath

import (
	"crypto/subtle"
	"encoding/binary"

	"chain/crypto/ed25519/internal/edwards25519"
)

// Scalar is a 256-bit little-endian scalar.
type Scalar [32]byte

var (
	// Zero is the number 0.
	Zero Scalar

	// One is the number 1.
	One = Scalar{1}

	// NegOne is the number -1 mod L
	NegOne = Scalar{
		0xec, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
		0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
	}

	// L is the subgroup order:
	// 2^252 + 27742317777372353535851937790883648493
	L = Scalar{
		0xed, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
		0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
	}
)

func (s *Scalar) SetUint64(n uint64) {
	*s = Zero
	binary.LittleEndian.PutUint64(s[:8], n)
}

// Add computes x+y (mod L) and places the result in z, returning
// that. Any or all of x, y, and z may be the same pointer.
func (z *Scalar) Add(x, y *Scalar) *Scalar {
	return z.MulAdd(x, &One, y)
}

// Sub computes x-y (mod L) and places the result in z, returning
// that. Any or all of x, y, and z may be the same pointer.
func (z *Scalar) Sub(x, y *Scalar) *Scalar {
	return z.MulAdd(y, &NegOne, x)
}

// Neg negates x (mod L) and places the result in z, returning that. X
// and z may be the same pointer.
func (z *Scalar) Neg(x *Scalar) *Scalar {
	return z.MulAdd(x, &NegOne, &Zero)
}

// MulAdd computes ab+c (mod L) and places the result in z, returning
// that. Any or all of the pointers may be the same.
func (z *Scalar) MulAdd(a, b, c *Scalar) *Scalar {
	edwards25519.ScMulAdd((*[32]byte)(z), (*[32]byte)(a), (*[32]byte)(b), (*[32]byte)(c))
	return z
}

func (z *Scalar) Equal(x *Scalar) bool {
	return subtle.ConstantTimeCompare(x[:], z[:]) == 1
}

// Prune performs the pruning operation in-place.
func (z *Scalar) Prune() {
	z[0] &= 248
	z[31] &= 127
	z[31] |= 64
}

// Reduce takes a 512-bit scalar and reduces it mod L, placing the
// result in z and returning that.
func (z *Scalar) Reduce(x *[64]byte) *Scalar {
	edwards25519.ScReduce((*[32]byte)(z), x)
	return z
}
