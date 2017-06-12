package sha3

import "io"

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

func newTupleHash(securitybits int, tuple [][]byte, s []byte) (d *state) {
	d = newCShake(securitybits, []byte("TupleHash"), s)
	for _, item := range tuple {
		encodeString(d, item)
	}
	return d
}
