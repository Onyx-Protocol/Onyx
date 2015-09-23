package bc

import (
	"encoding/hex"
	"errors"
)

// ID encodes hash into a byte-reversed hex-encoded string.
func ID(hash []byte) string {
	b := make([]byte, len(hash))
	copy(b, hash)
	for i := 0; i < len(b)/2; i++ {
		b[i], b[len(b)-1-i] = b[len(b)-1-i], b[i]
	}
	return hex.EncodeToString(b)
}

// DecodeHash256 decodes a [32]byte hash from an ID string.
// The string should be the hexadecimal string of a byte-reversed hash,
// but any missing characters
// result in zero padding at the end of the [32]byte.
func DecodeHash256(id string, hash *[32]byte) error {
	// Return error if hash string is too long.
	if len(id) > 32*2 {
		return errors.New("invalid string")
	}

	// Hex decoder expects the hash to be a multiple of two.
	if len(id)%2 != 0 {
		id = "0" + id
	}

	// Convert string hash to bytes.
	buf, err := hex.DecodeString(id)
	if err != nil {
		return err
	}

	// Un-reverse the decoded bytes, copying into in leading bytes of a
	// [32]byte.  There is no need to explicitly pad the result as any
	// missing (when len(buf) < HashSize) bytes from the decoded hex string
	// will remain zeros at the end of the [32]byte.
	for i := range buf {
		hash[i] = buf[len(buf)-1-i]
	}
	return nil
}
