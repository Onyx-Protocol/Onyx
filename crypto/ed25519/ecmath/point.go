package ecmath

import (
	"crypto/subtle"

	"chain/crypto/ed25519/internal/edwards25519"
)

// Point is a point on the ed25519 curve.
type Point edwards25519.ExtendedGroupElement

// ZeroPoint is the zero point on the ed25519 curve (not the zero value of Point).
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
func (z *Point) ScMul(x *Point, y *Scalar) *Point {
	return z.ScMulAdd(x, y, &Zero)
}

// ScMulBase multiplies the ed25519 base point by x and places the
// result in z, returning that.
func (z *Point) ScMulBase(x *Scalar) *Point {
	edwards25519.GeScalarMultBase((*edwards25519.ExtendedGroupElement)(z), (*[32]byte)(x))
	return z
}

// ScMulAdd computes xa+yB, where B is the ed25519 base point, and
// places the result in z, returning that.
func (z *Point) ScMulAdd(a *Point, x, y *Scalar) *Point {
	// TODO: replace with constant-time implementation to avoid
	// sidechannel attacks

	var p edwards25519.ProjectiveGroupElement
	edwards25519.GeDoubleScalarMultVartime(&p, (*[32]byte)(x), (*edwards25519.ExtendedGroupElement)(a), (*[32]byte)(y))

	var buf [32]byte
	p.ToBytes(&buf)
	// TODO(bobg): double-check that it's OK to ignore the return value
	// from ExtendedGroupElement.FromBytes here. (It's a bool indicating
	// that its input represented a legal value.)
	(*edwards25519.ExtendedGroupElement)(z).FromBytes(&buf)
	return z
}

func (z *Point) Encode() [32]byte {
	var e [32]byte
	(*edwards25519.ExtendedGroupElement)(z).ToBytes(&e)
	return e
}

func (z *Point) Decode(e [32]byte) (*Point, bool) {
	ok := (*edwards25519.ExtendedGroupElement)(z).FromBytes(&e)
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
