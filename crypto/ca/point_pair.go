package ca

import (
	"chain/crypto/ed25519/ecmath"
	"crypto/subtle"
)

// PointPair is an ordered pair of points on the ed25519 curve.
type PointPair struct {
	Point1 ecmath.Point
	Point2 ecmath.Point
}

// ZeroPointPair is a pair of identity elements on the ed25519 group (not all-zero points).
var ZeroPointPair PointPair

// Add adds the point pairs X and Y, storing the result in Z and
// returning that. Any or all of X, Y, and Z may be the same pointers.
func (z *PointPair) Add(x, y *PointPair) *PointPair {
	z.Point1.Add(&x.Point1, &y.Point1)
	z.Point2.Add(&x.Point2, &y.Point2)
	return z
}

// Sub subtracts Y from X, storing the result in Z and
// returning that. Any or all of X, Y, and Z may be the same pointers.
func (z *PointPair) Sub(x, y *PointPair) *PointPair {
	z.Point1.Sub(&x.Point1, &y.Point1)
	z.Point2.Sub(&x.Point2, &y.Point2)
	return z
}

// ScMul multiplies the EC point pair X by the scalar Y, placing the result
// in Z and returning that. X and Z may be the same pointer.
func (z *PointPair) ScMul(x *PointPair, y *ecmath.Scalar) *PointPair {
	z.Point1.ScMul(&x.Point1, y)
	z.Point2.ScMul(&x.Point2, y)
	return z
}

// Encode encodes the point pair as a 64-byte binary string.
func (z *PointPair) Encode() (result [64]byte) {
	var buf32 [32]byte
	buf32 = z.Point1.Encode()
	copy(result[:32], buf32[:])
	buf32 = z.Point2.Encode()
	copy(result[32:], buf32[:])
	return
}

// Decode instantiates a point pair from a 64-byte binary string.
func (z *PointPair) Decode(e [64]byte) (*PointPair, bool) {
	var buf32 [32]byte
	var ok bool
	copy(buf32[:], e[0:32])
	_, ok = z.Point1.Decode(buf32)
	if !ok {
		return z, false
	}
	copy(buf32[:], e[32:64])
	_, ok = z.Point2.Decode(buf32)
	if !ok {
		return z, false
	}
	return z, true
}

// ConstTimeEqual compares two point pairs in constant time
func (z *PointPair) ConstTimeEqual(x *PointPair) bool {
	xe := x.Encode()
	ze := z.Encode()
	return subtle.ConstantTimeCompare(xe[:], ze[:]) == 1
}

func init() {
	ZeroPointPair.Point1 = ecmath.ZeroPoint
	ZeroPointPair.Point2 = ecmath.ZeroPoint
}
