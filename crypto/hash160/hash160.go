// Package hash160 implements the Hash160 hash algorithm
// as defined in the Bitcoin Core source code.
//
// Sum(data) is ripemd160(sha256(data)).
package hash160

import (
	"crypto/sha256"
	"hash"

	"golang.org/x/crypto/ripemd160"
)

// BlockSize is the block size of Hash160 in bytes.
const BlockSize = sha256.BlockSize

// Size is the size of a Hash160 checksum in bytes.
const Size = ripemd160.Size

// New returns a new hash.Hash computing the Hash160 checksum.
func New() hash.Hash {
	return &digest{sha256.New(), ripemd160.New()}
}

type digest struct {
	inner hash.Hash // sha256.digest
	outer hash.Hash // ripemd160.digest
}

func (d *digest) Reset()         { d.inner.Reset() }
func (d *digest) Size() int      { return Size }
func (d *digest) BlockSize() int { return BlockSize }
func (d *digest) Write(p []byte) (int, error) {
	return d.inner.Write(p)
}

func (d *digest) Sum(in []byte) []byte {
	inner := d.inner.Sum(nil)
	d.outer.Reset()
	d.outer.Write(inner[:])
	return d.outer.Sum(in)
}

// Sum returns the Hash160 checksum of the data.
func Sum(data []byte) [Size]byte {
	var sum [Size]byte
	h := New()
	h.Write(data)
	h.Sum(sum[:0])
	return sum
}
