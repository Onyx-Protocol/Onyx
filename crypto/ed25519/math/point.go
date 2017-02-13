package math

import (
	"crypto/subtle"

	"chain/crypto/ed25519/internal/edwards25519"
)

type (
	// Point is a point on the ed25519 curve.
	Point edwards25519.ExtendedGroupElement

	// EncodedPoint is a serialized representation of a point.
	EncodedPoint [32]byte
)

// ZeroPoint is the zero point on the ed25519 curve.
var ZeroPoint Point

// Add adds the points in x and y, storing the result in z and
// returning that. Any or all of x, y, and z may be the same pointers.
func (z *Point) Add(x, y *Point) *Point {
	var y2 edwards25519.CachedGroupElement
	(*edwards25519.ExtendedGroupElement)(y).ToCached(&y2)

	var z2 edwards25519.CompletedGroupElement
	edwards25519.GeAdd(&z2, (*edwards25519.ExtendedGroupElement)(x), &y2)

	z2.ToExtended((*edwards25519.ExtendedGroupElement)(z))
	return z
}

// Sub subtracts y from x, storing the result in z and
// returning that. Any or all of x, y, and z may be the same pointers.
func (z *Point) Sub(x, y *Point) *Point {
	var y2 edwards25519.CachedGroupElement
	(*edwards25519.ExtendedGroupElement)(y).ToCached(&y2)

	var z2 edwards25519.CompletedGroupElement
	edwards25519.GeSub(&z2, (*edwards25519.ExtendedGroupElement)(x), &y2)

	z2.ToExtended((*edwards25519.ExtendedGroupElement)(z))
	return z
}

// ScMul multiplies the EC point x by the scalar y, placing the result
// in z and returning that. X and z may be the same pointer.
func (z *Point) ScMul(x *Point, y *Uint256le) *Point {
	return z.ScMulAdd(x, y, &Zero)
}

// ScMulBase multiplies the ed25519 base point by x and places the
// result in z, returning that.
func (z *Point) ScMulBase(x *Uint256le) *Point {
	edwards25519.GeScalarMultBase((*edwards25519.ExtendedGroupElement)(z), (*[32]byte)(x))
	return z
}

// ScMulAdd computes xa+yB, where B is the ed25519 base point, and
// places the result in z, returning that.
func (z *Point) ScMulAdd(a *Point, x, y *Uint256le) *Point {
	// TODO: replace with constant-time implementation to avoid
	// sidechannel attacks

	var p edwards25519.ProjectiveGroupElement
	edwards25519.GeDoubleScalarMultVartime(&p, (*[32]byte)(x), (*edwards25519.ExtendedGroupElement)(a), (*[32]byte)(y))

	var buf [32]byte
	p.ToBytes(&buf)
	(*edwards25519.ExtendedGroupElement)(z).FromBytes(&buf) // xxx don't need to check return val, right?
	return z
}

func (z *Point) Encode() (e EncodedPoint) {
	(*edwards25519.ExtendedGroupElement)(z).ToBytes((*[32]byte)(&e))
	return e
}

func (z *Point) Decode(e EncodedPoint) (*Point, bool) {
	ok := (*edwards25519.ExtendedGroupElement)(z).FromBytes((*[32]byte)(&e))
	return z, ok
}

func (z *Point) ConstTimeEqual(x *Point) bool {
	xe := x.Encode()
	ze := z.Encode()
	return subtle.ConstantTimeCompare(xe[:], ze[:]) == 1
}

func init() {
	(*edwards25519.ExtendedGroupElement)(&ZeroPoint).Zero()
}
