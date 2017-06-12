package sha3

import "encoding/binary"

// NewCShake128 creates a new cSHAKE128 variable-output-length customizable ShakeHash.
// Its generic security strength is 128 bits against all attacks
// if at least 32 bytes of its output are used.
// n is a customization string for derived functions specified by NIST.
// Set n to an empty string if you are building your own derived function.
// s is a user-defined customization string.
func NewCShake128(n []byte, s []byte) ShakeHash {
	return newCShake(128, n, s)
}

// NewCShake256 creates a new cSHAKE256 variable-output-length customizable ShakeHash.
// Its generic security strength is 256 bits against all attacks
// if at least 64 bytes of its output are used.
// n is a customization string for derived functions specified by NIST.
// Set n to an empty string if you are building your own derived function.
// s is a user-defined customization string.
func NewCShake256(n []byte, s []byte) ShakeHash {
	return newCShake(256, n, s)
}

// CShakeSum128 writes an arbitrary-length digest of data into hash.
// n is a customization string for derived functions specified by NIST.
// Set n to an empty string if you are building your own derived function.
// s is a user-defined customization string.
func CShakeSum128(hash, data, n, s []byte) {
	h := NewCShake128(n, s)
	h.Write(data)
	h.Read(hash)
}

// CShakeSum256 writes an arbitrary-length digest of data into hash.
// n is a customization string for derived functions specified by NIST.
// Set n to an empty string if you are building your own derived function.
// s is a user-defined customization string.
func CShakeSum256(hash, data, n, s []byte) {
	h := NewCShake256(n, s)
	h.Write(data)
	h.Read(hash)
}

var zero [maxRate]byte

func newCShake(securitybits int, n []byte, s []byte) (d *state) {
	if len(n) == 0 && len(s) == 0 {
		if securitybits == 128 {
			return &state{rate: 168, dsbyte: 0x1f} // regular SHAKE-128
		} else if securitybits == 256 {
			return &state{rate: 136, dsbyte: 0x1f} // regular SHAKE-256
		} else {
			panic("invalid security level for cSHAKE")
		}
	}
	if securitybits == 128 {
		d = &state{rate: 168, dsbyte: 0x04}
	} else if securitybits == 256 {
		d = &state{rate: 136, dsbyte: 0x04}
	} else {
		panic("invalid security level for cSHAKE")
	}
	d.initCShake(n, s)
	return d
}

// The initialization of cSHAKE
func (d *state) initCShake(n []byte, s []byte) {
	var c int
	c += leftEncode(d, uint64(d.rate))
	c += encodeString(d, n)
	c += encodeString(d, s)
	d.Write(zero[:d.rate-(c%d.rate)])
}

func encodeString(d *state, s []byte) int {
	n := leftEncode(d, uint64(len(s)*8))
	w, _ := d.Write(s)
	return n + w
}

// rightEncode encodes integer in a variable-length encoding
// unambiguously parseable from the end of a string.
// Used by TupleHash and KMAC.
func rightEncode(d *state, value uint64) int {
	input := d.varintbuf[:]
	copy(input, zero[:])
	var offset uint
	if value == 0 {
		offset = 7
	} else {
		binary.BigEndian.PutUint64(input[0:], value)
		for offset = 0; offset < 8; offset++ {
			if input[offset] != 0 {
				break
			}
		}
	}
	input[8] = byte(8 - offset)
	b := input[offset:]
	d.Write(b)
	return len(b)
}

// leftEncode encodes integer in a variable-length encoding
// unambiguously parseable from the beginning of a string.
// Used to encode strings for all cSHAKE functions.
func leftEncode(d *state, value uint64) int {
	input := d.varintbuf[:]
	copy(input, zero[:])
	var offset uint
	if value == 0 {
		offset = 8
	} else {
		binary.BigEndian.PutUint64(input[1:], value)
		for offset = 0; offset < 9; offset++ {
			if input[offset] != 0 {
				break
			}
		}
	}
	input[offset-1] = byte(9 - offset)
	b := input[offset-1:]
	d.Write(b)
	return len(b)
}
