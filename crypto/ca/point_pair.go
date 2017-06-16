package ca

import (
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"

	"chain/crypto/ed25519/ecmath"
)

// PointPair is an ordered pair of points on the ed25519 curve.
type PointPair [2]ecmath.Point

// ZeroPointPair is a pair of identity elements on the ed25519 group (not all-zero points).
var ZeroPointPair PointPair

// Add adds the point pairs X and Y, storing the result in Z and
// returning that. Any or all of X, Y, and Z may be the same pointers.
func (z *PointPair) Add(x, y *PointPair) *PointPair {
	z[0].Add(&x[0], &y[0])
	z[1].Add(&x[1], &y[1])
	return z
}

// Sub subtracts Y from X, storing the result in Z and
// returning that. Any or all of X, Y, and Z may be the same pointers.
func (z *PointPair) Sub(x, y *PointPair) *PointPair {
	z[0].Sub(&x[0], &y[0])
	z[1].Sub(&x[1], &y[1])
	return z
}

// ScMul multiplies the EC point pair X by the scalar Y, placing the result
// in Z and returning that. X and Z may be the same pointer.
func (z *PointPair) ScMul(x *PointPair, y *ecmath.Scalar) *PointPair {
	z[0].ScMul(&x[0], y)
	z[1].ScMul(&x[1], y)
	return z
}

// Encode encodes the point pair as a 64-byte binary string.
func (z *PointPair) Encode() (result [64]byte) {
	var buf32 [32]byte
	buf32 = z[0].Encode()
	copy(result[:32], buf32[:])
	buf32 = z[1].Encode()
	copy(result[32:], buf32[:])
	return
}

// Decode instantiates a point pair from a 64-byte binary string.
func (z *PointPair) Decode(e [64]byte) (*PointPair, bool) {
	var buf32 [32]byte
	var ok bool
	copy(buf32[:], e[0:32])
	_, ok = z[0].Decode(buf32)
	if !ok {
		return z, false
	}
	copy(buf32[:], e[32:64])
	_, ok = z[1].Decode(buf32)
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

// Bytes returns binary representation of a point pair (64-byte slice).
func (p *PointPair) Bytes() []byte {
	return append(p[0].Bytes(), p[1].Bytes()...)
}

// String returns hex representation of a point pair.
func (p *PointPair) String() string {
	return hex.EncodeToString(p.Bytes())
}

// MarshalBinary encodes the point pair into a binary form and returns the result (32-byte slice).
func (p *PointPair) MarshalBinary() ([]byte, error) {
	return p.Bytes(), nil
}

// UnmarshalBinary decodes a point pair for a given slice.
// Returns error if the slice is not 32-byte long or the encoding is invalid.
func (z *PointPair) UnmarshalBinary(data []byte) error {
	if len(data) != 64 {
		return fmt.Errorf("invalid size of the encoded ca.PointPair: %d bytes (must be 64)", len(data))
	}
	var err error
	err = z[0].UnmarshalBinary(data[0:32])
	if err != nil {
		return err
	}
	err = z[1].UnmarshalBinary(data[32:64])
	return err
}

// WriteTo writes 32-byte encoding of a point pair.
func (z *PointPair) WriteTo(w io.Writer) (n int64, err error) {
	n1, err := z[0].WriteTo(w)
	if err != nil {
		return n1, err
	}
	n2, err := z[1].WriteTo(w)
	return n1 + n2, err
}

// ReadFrom attempts to read 32 bytes and decode a point pair.
func (z *PointPair) ReadFrom(r io.Reader) (n int64, err error) {
	n1, err := z[0].ReadFrom(r)
	if err != nil {
		return n1, err
	}
	n2, err := z[1].ReadFrom(r)
	return n1 + n2, err
}

// MarshalText returns a hex-encoded point pair.
func (z *PointPair) MarshalText() ([]byte, error) {
	b1, _ := z[0].MarshalText()
	b2, _ := z[1].MarshalText()
	return append(b1, b2...), nil
}

// UnmarshalText decodes a point pair from a hex-encoded buffer.
func (z *PointPair) UnmarshalText(data []byte) error {
	if len(data) != hex.EncodedLen(64) {
		return fmt.Errorf("ca.PointPair.UnmarshalText got input with wrong length %d", len(data))
	}
	var err error
	err = z[0].UnmarshalText(data[0:hex.EncodedLen(32)])
	if err != nil {
		return err
	}
	err = z[1].UnmarshalText(data[hex.EncodedLen(32):hex.EncodedLen(64)])
	return err
}

func init() {
	ZeroPointPair[0] = ecmath.ZeroPoint
	ZeroPointPair[1] = ecmath.ZeroPoint
}
