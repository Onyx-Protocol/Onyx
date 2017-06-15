package ecmath

import (
	"encoding/hex"
	"fmt"
	"io"

	"chain/crypto/ed25519/internal/edwards25519"
)

// Bytes returns binary representation of a EC point (32-byte slice)
func (p *Point) Bytes() []byte {
	var buf [32]byte
	(*edwards25519.ExtendedGroupElement)(p).ToBytes(&buf)
	return buf[:]
}

// String returns hex representation of a EC point
func (p *Point) String() string {
	return hex.EncodeToString(p.Bytes())
}

// MarshalBinary encodes the receiver into a binary form and returns the result (32-byte slice).
func (p *Point) MarshalBinary() ([]byte, error) {
	return p.Bytes(), nil
}

// UnmarshalBinary decodes point for a given slice.
// Returns error if the slice is not 32-byte long or the encoding is invalid.
func (p *Point) UnmarshalBinary(data []byte) error {
	var buf [32]byte
	if len(data) != 32 {
		return fmt.Errorf("invalid size of the encoded ecmath.Point: %d bytes (must be 32)", len(data))
	}
	copy(buf[:], data)
	if !(*edwards25519.ExtendedGroupElement)(p).FromBytes(&buf) {
		return fmt.Errorf("invalid ecmath.Point encoding")
	}
	return nil
}

// WriteTo writes 32-byte encoding of a point.
func (p *Point) WriteTo(w io.Writer) (n int64, err error) {
	m, err := w.Write(p.Bytes())
	return int64(m), err
}

// ReadFrom attempts to read 32 bytes and decode a point.
func (p *Point) ReadFrom(r io.Reader) (n int64, err error) {
	var b [32]byte
	m, err := io.ReadFull(r, b[:])
	if err != nil {
		return
	}
	err = p.UnmarshalBinary(b[:])
	return int64(m), err
}

// MarshalText returns a hex-encoded point.
func (p *Point) MarshalText() ([]byte, error) {
	buf := p.Bytes()
	res := make([]byte, hex.EncodedLen(len(buf)))
	hex.Encode(res, buf)
	return res, nil
}

// UnmarshalText decodes a point from a hex-encoded buffer.
func (p *Point) UnmarshalText(b []byte) error {
	var buf [32]byte
	if len(b) != hex.EncodedLen(len(buf)) {
		return fmt.Errorf("ecmath.Point.UnmarshalText got input with wrong length %d", len(b))
	}
	_, err := hex.Decode(buf[:], b)
	if err != nil {
		return err
	}
	return p.UnmarshalBinary(buf[:])
}
