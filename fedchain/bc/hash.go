package bc

import (
	"chain/errors"
	"encoding/hex"
	"fmt"
)

// Hash represents a 256-bit hash
type Hash [32]byte

// String returns the bytes of h encoded in hex.
func (h Hash) String() string {
	b, _ := h.MarshalText()
	return string(b)
}

// MarshalText satisfies the TextMarshaler interface.
// It returns the bytes of h encoded in hex,
// for formats that can't hold arbitrary binary data.
// It never returns an error.
func (h Hash) MarshalText() ([]byte, error) {
	b := make([]byte, hex.EncodedLen(len(h)))
	hex.Encode(b, h[:])
	return b, nil
}

// UnmarshalText satisfies the TextUnmarshaler interface.
// It decodes hex data from b into h.
func (h *Hash) UnmarshalText(b []byte) error {
	if len(b) != hex.EncodedLen(len(h)) {
		return fmt.Errorf("bad hash hex length %d", len(b))
	}
	_, err := hex.Decode(h[:], b)
	return err
}

// ParseHash takes a hex-encoded hash and returns
// a 32 byte array.
func ParseHash(s string) (h Hash, err error) {
	if len(s) != hex.EncodedLen(len(h)) {
		return h, errors.New("wrong hex length")
	}
	_, err = hex.Decode(h[:], []byte(s))
	return h, errors.Wrap(err, "decode hex")
}
