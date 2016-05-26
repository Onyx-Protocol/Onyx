// Copyright (c) 2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import "math"

const (
	// maxScriptNumLen is the maximum number of bytes data being interpreted
	// as an integer may be.
	maxScriptNumLen = 8
)

// scriptNum represents a numeric value used in the scripting engine with
// special handling to deal with the subtle semantics required by consensus.
//
// All numbers are stored on the data and alternate stacks encoded as little
// endian with a sign bit.  All numeric opcodes such as OP_ADD, OP_SUB,
// and OP_MUL, are only allowed to operate on 8-byte integers. Each numeric
// opcode has its own limits on its operands to prevent overflow.
//
// This type stores all integers as an int64. It is the responsibility of the
// individual opcodes to perform bounds checking to prevent overflow.
type scriptNum int64

// checkMinimalDataEncoding returns whether or not the passed byte array adheres
// to the minimal encoding requirements.
func checkMinimalDataEncoding(v []byte) error {
	if len(v) == 0 {
		return nil
	}

	// Check that the number is encoded with the minimum possible
	// number of bytes.
	//
	// If the most-significant-byte - excluding the sign bit - is zero
	// then we're not minimal.  Note how this test also rejects the
	// negative-zero encoding, [0x80].
	if v[len(v)-1]&0x7f == 0 {
		// One exception: if there's more than one byte and the most
		// significant bit of the second-most-significant-byte is set
		// it would conflict with the sign bit.  An example of this case
		// is +-255, which encode to 0xff00 and 0xff80 respectively.
		// (big-endian).
		if len(v) == 1 || v[len(v)-2]&0x80 == 0 {
			return ErrStackMinimalData
		}
	}
	return nil
}

// Bytes returns the number serialized as a little endian with a sign bit.
//
// Example encodings:
//       127 -> [0x7f]
//      -127 -> [0xff]
//       128 -> [0x80 0x00]
//      -128 -> [0x80 0x80]
//       129 -> [0x81 0x00]
//      -129 -> [0x81 0x80]
//       256 -> [0x00 0x01]
//      -256 -> [0x00 0x81]
//     32767 -> [0xff 0x7f]
//    -32767 -> [0xff 0xff]
//     32768 -> [0x00 0x80 0x00]
//    -32768 -> [0x00 0x80 0x80]
func (n scriptNum) Bytes() []byte {
	// Zero encodes as an empty byte slice.
	if n == 0 {
		return nil
	}

	// Take the absolute value and keep track of whether it was originally
	// negative.
	isNegative := n < 0
	if isNegative {
		n = -n
	}

	// Encode to little endian.  The maximum number of encoded bytes is 9
	// (8 bytes for max int64 plus a potential byte for sign extension).
	result := make([]byte, 0, 9)
	for n > 0 {
		result = append(result, byte(n&0xff))
		n >>= 8
	}

	// When the most significant byte already has the high bit set, an
	// additional high byte is required to indicate whether the number is
	// negative or positive.  The additional byte is removed when converting
	// back to an integral and its high bit is used to denote the sign.
	//
	// Otherwise, when the most significant byte does not already have the
	// high bit set, use it to indicate the value is negative, if needed.
	if result[len(result)-1]&0x80 != 0 {
		extraByte := byte(0x00)
		if isNegative {
			extraByte = 0x80
		}
		result = append(result, extraByte)

	} else if isNegative {
		result[len(result)-1] |= 0x80
	}

	return result
}

func Int64ToScriptBytes(n int64) []byte {
	return scriptNum(n).Bytes()
}

func BoolToScriptBytes(b bool) []byte {
	if b {
		return Int64ToScriptBytes(1)
	}
	return Int64ToScriptBytes(0)
}

// Int32 returns the script number clamped to a valid int32.
func (n scriptNum) Int32() int32 {
	if n > math.MaxInt32 {
		return math.MaxInt32
	}

	if n < math.MinInt32 {
		return math.MinInt32
	}

	return int32(n)
}

// makeScriptNum interprets the passed serialized bytes as an encoded integer
// and returns the result as a script number.
//
// Since the consensus rules dictate the serialized bytes interpreted as ints
// are only allowed to be in the range [-2^31 + 1, 2^31 - 1], an error will be
// returned when the provided bytes would result in a number outside of that
// range.
//
// The requireMinimal flag causes an error to be returned if additional checks
// on the encoding determine it is not represented with the smallest possible
// number of bytes or is the negative 0 encoding, [0x80].  For example, consider
// the number 127.  It could be encoded as [0x7f], [0x7f 0x00],
// [0x7f 0x00 0x00 ...], etc.  All forms except [0x7f] will return an error with
// requireMinimal enabled.
//
// See the Bytes function documentation for example encodings.
func makeScriptNum(v []byte, requireMinimal bool) (scriptNum, error) {
	return MakeScriptNumWithMaxLen(v, requireMinimal, maxScriptNumLen)
}

func MakeScriptNumWithMaxLen(v []byte, requireMinimal bool, maxLen int) (scriptNum, error) {
	if len(v) > maxLen {
		return 0, ErrStackNumberTooBig
	}

	// Enforce minimal encoded if requested.
	if requireMinimal {
		if err := checkMinimalDataEncoding(v); err != nil {
			return 0, err
		}
	}

	// Zero is encoded as an empty byte slice.
	if len(v) == 0 {
		return 0, nil
	}

	// Decode from little endian.
	var result int64
	for i, val := range v {
		result |= int64(val) << uint8(8*i)
	}

	// When the most significant byte of the input bytes has the sign bit
	// set, the result is negative.  So, remove the sign bit from the result
	// and make it negative.
	if v[len(v)-1]&0x80 != 0 {
		// The maximum length of v has already been determined to be 4
		// above, so uint8 is enough to cover the max possible shift
		// value of 24.
		result &= ^(int64(0x80) << uint8(8*(len(v)-1)))
		return scriptNum(-result), nil
	}

	return scriptNum(result), nil
}
