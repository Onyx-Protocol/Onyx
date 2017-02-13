package math

import (
	"chain-stealth/crypto/ed25519/edwards25519"
	"crypto/subtle"
)

// Uint256le is a 256-bit little-endian scalar.
type Uint256le [32]byte

var (
	// Zero is the number 0.
	Zero Uint256le

	// One is the number 1.
	One = Uint256le{1}

	// NegOne is the number -1 mod L
	NegOne = Uint256le{
		0xec, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
		0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
	}

	// L is the subgroup order:
	// 2^252 + 27742317777372353535851937790883648493
	L = Uint256le{
		0xed, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
		0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
	}
)

// Add computes x+y (mod L) and places the result in z, returning
// that. Any or all of x, y, and z may be the same pointer.
func (z *Uint256le) Add(x, y *Uint256le) *Uint256le {
	return z.MulAdd(x, &One, y)
}

// Sub computes x-y (mod L) and places the result in z, returning
// that. Any or all of x, y, and z may be the same pointer.
func (z *Uint256le) Sub(x, y *Uint256le) *Uint256le {
	return z.MulAdd(y, &NegOne, x)
}

// Neg negates x (mod L) and places the result in z, returning that. X
// and z may be the same pointer.
func (z *Uint256le) Neg(x *Uint256le) *Uint256le {
	return z.MulAdd(x, &NegOne, &Zero)
}

// MulAdd computes ab+c (mod L) and places the result in z, returning
// that. Any or all of the pointers may be the same.
func (z *Uint256le) MulAdd(a, b, c *Uint256le) *Uint256le {
	edwards25519.ScMulAdd((*[32]byte)(z), (*[32]byte)(a), (*[32]byte)(b), (*[32]byte)(c))
	return z
}

func (z *Uint256le) Equal(x *Uint256le) bool {
	return subtle.ConstantTimeCompare(x[:], z[:]) == 1
}

// Prune performs the pruning operation in-place.
func (z *Uint256le) Prune() {
	z[0] &= 248
	z[31] &= 127
	z[31] |= 64
}

// Reduce takes a 512-bit scalar and reduces it mod L, placing the
// result in z and returning that.
func (z *Uint256le) Reduce(x *[64]byte) *Uint256le {
	edwards25519.ScReduce((*[32]byte)(z), x)
	return z
}
