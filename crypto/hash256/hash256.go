// Package hash256 implements the Hash256 hash algorithm
// as defined in the Bitcoin Core source code.
//
// Sum(data) is sha256(sha256(data)).
package hash256

import (
	"crypto/sha256"
	"hash"
)

// BlockSize is the block size of Hash256 in bytes.
const BlockSize = sha256.BlockSize

// Size is the size of a Hash256 checksum in bytes.
const Size = sha256.Size

// New returns a new hash.Hash computing the Hash256 checksum.
func New() hash.Hash {
	return &digest{sha256.New(), sha256.New()}
}

type digest struct {
	inner hash.Hash // sha256.digest
	outer hash.Hash // sha256.digest
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

// Sum returns the Hash256 checksum of the data.
func Sum(data []byte) [Size]byte {
	inner := sha256.Sum256(data)
	return sha256.Sum256(inner[:])
}
