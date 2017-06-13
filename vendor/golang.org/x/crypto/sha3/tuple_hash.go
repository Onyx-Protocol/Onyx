package sha3

import (
	"hash"
	"io"
)

type thash struct { // implements hash.Hash and ShakeHash
	d             *state
	lengthEmitted bool
}

// TupleHash128 hashes a tuple with a given customization string s.
// Output is written to `out`. len(out) determines the output size.
func TupleHash128(tuple [][]byte, s []byte, out []byte) {
	shake := newTupleHash(128, tuple, s)
	rightEncode(shake, uint64(len(out)*8))
	shake.Read(out)
}

// TupleHash256 hashes a tuple with a given customization string s.
// Output is written to `out`. len(out) determines the output size.
func TupleHash256(tuple [][]byte, s []byte, out []byte) {
	shake := newTupleHash(256, tuple, s)
	rightEncode(shake, uint64(len(out)*8))
	shake.Read(out)
}

// TupleHashXOF128 provides an arbitrary-length output.
func TupleHashXOF128(tuple [][]byte, s []byte) io.Reader {
	shake := newTupleHash(128, tuple, s)
	rightEncode(shake, 0)
	return shake
}

// TupleHashXOF256 provides an arbitrary-length output.
func TupleHashXOF256(tuple [][]byte, s []byte) io.Reader {
	shake := newTupleHash(256, tuple, s)
	rightEncode(shake, 0)
	return shake
}

// NewTupleHash128 creates an instance of Hash with a given key,
// output length in bytes and a customization string s.
func NewTupleHash128(length int, s []byte) hash.Hash {
	return newTupleHasher(128, length, s)
}

// NewTupleHash256 creates an instance of Hash with a given key,
// output length in bytes and a customization string s.
func NewTupleHash256(length int, s []byte) hash.Hash {
	return newTupleHasher(256, length, s)
}

// NewTupleHashXOF128 provides an arbitrary-length output.
func NewTupleHashXOF128(s []byte) ShakeHash {
	return newTupleHasher(128, 0, s)
}

// NewTupleHashXOF256 provides an arbitrary-length output.
func NewTupleHashXOF256(s []byte) ShakeHash {
	return newTupleHasher(256, 0, s)
}

// BlockSize returns the rate of sponge underlying this hash function.
func (t *thash) BlockSize() int { return t.d.rate }

// Size returns the output size of the hash function in bytes.
func (t *thash) Size() int { return t.d.outputLen }

func (t *thash) Reset() {
	t.lengthEmitted = false
	t.d.Reset()
}

func (t *thash) Clone() ShakeHash {
	return t.clone()
}

// Write writes a tuple item with necessary length prefix.
// If you need to write several chunks of one item, buffer them first
// in a single slice and then pass it to the Write method.
func (t *thash) Write(p []byte) (written int, err error) {
	written = encodeString(t.d, p)
	return
}

func (t *thash) Read(out []byte) (n int, err error) {
	n = 0
	if !t.lengthEmitted {
		n = t.encodeOutputLength()
		t.lengthEmitted = true
	}
	m, err := t.d.Read(out)
	return n + m, err
}

// Sum applies padding to the hash state and then squeezes out the desired
// number of output bytes.
func (t *thash) Sum(in []byte) []byte {
	// Make a copy of the original hash so that caller can keep writing
	// and summing.
	dup := t.clone()
	hash := make([]byte, dup.d.outputLen)
	dup.Read(hash)
	return append(in, hash...)
}

func newTupleHash(securitybits int, tuple [][]byte, s []byte) (d *state) {
	d = newCShake(securitybits, []byte("TupleHash"), s)
	for _, item := range tuple {
		encodeString(d, item)
	}
	return d
}

func newTupleHasher(securitybits int, length int, s []byte) *thash {
	t := thash{d: newCShake(securitybits, []byte("TupleHash"), s)}
	t.d.outputLen = length
	return &t
}

func (t *thash) encodeOutputLength() int {
	return rightEncode(t.d, uint64(8*t.d.outputLen))
}

func (t *thash) clone() *thash {
	t2 := *t
	t2.d = t2.d.clone()
	return &t2
}
