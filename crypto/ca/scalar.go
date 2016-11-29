package ca

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"chain-stealth/crypto/ed25519/edwards25519"
)

type Scalar [32]byte

var (
	cofactor   = Scalar{8}
	one        = Scalar{1}
	ZeroScalar = Scalar{0}

	// subgroup order: 2^252 + 27742317777372353535851937790883648493
	order = Scalar{
		0xed, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
		0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
	}

	// subgroup order - 1.
	negone = Scalar{
		0xec, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
		0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
	}
)

func (s Scalar) MarshalText() ([]byte, error) {
	b := make([]byte, hex.EncodedLen(len(s)))
	hex.Encode(b, s[:])
	return b, nil
}

func (s *Scalar) UnmarshalText(b []byte) error {
	if len(b) != hex.EncodedLen(len(s)) {
		return fmt.Errorf("Scalar.UnmarshalText got input with wrong length %d", len(b))
	}
	_, err := hex.Decode(s[:], b)
	return err
}

func (a *Scalar) Add(b *Scalar) {
	edwards25519.ScMulAdd((*[32]byte)(a), (*[32]byte)(&one), (*[32]byte)(b), (*[32]byte)(a))
}

func (a *Scalar) sub(b *Scalar) {
	edwards25519.ScMulAdd((*[32]byte)(a), (*[32]byte)(&negone), (*[32]byte)(b), (*[32]byte)(a))
}

func (a *Scalar) negate() {
	edwards25519.ScMulAdd((*[32]byte)(a), (*[32]byte)(&negone), (*[32]byte)(a), (*[32]byte)(&ZeroScalar))
}

// a = ab + c
func (a *Scalar) mulAdd(b, c *Scalar) {
	edwards25519.ScMulAdd((*[32]byte)(a), (*[32]byte)(a), (*[32]byte)(b), (*[32]byte)(c))
}

func (a *Scalar) equal(b *Scalar) bool {
	return constTimeEqual(a[:], b[:])
}

func addScalars(a Scalar, b Scalar) Scalar {
	a.Add(&b)
	return a
}

func subScalars(a Scalar, b Scalar) Scalar {
	a.sub(&b)
	return a
}

func negateScalar(x Scalar) Scalar {
	x.negate()
	return x
}

// Returns (a*b + c) mod l
func multiplyAndAddScalars(a, b, c Scalar) Scalar {
	a.mulAdd(&b, &c)
	return a
}

func scalarFromUint64(value uint64) (result Scalar) {
	binary.LittleEndian.PutUint64(result[:8], value)
	return result
}

func pruneSecretScalar(scalar *[32]byte) {
	scalar[0] &= 248
	scalar[31] &= 127
	scalar[31] |= 64
}

func reducedScalar(longscalar [64]byte) (result Scalar) {
	edwards25519.ScReduce((*[32]byte)(&result), &longscalar)
	return result
}
