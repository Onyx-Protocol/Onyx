package bc

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/sha3"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/errors"
)

// Hash represents a 256-bit hash.  By convention, Hash objects are
// typically passed as values, not as pointers.
type Hash [32]byte

var EmptyStringHash = sha3.Sum256(nil)

// String returns the bytes of h encoded in hex.
func (h Hash) String() string {
	b, _ := h.MarshalText() // #nosec
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
		return errors.WithDetailf(
			fmt.Errorf("bad hash hex length %d", len(b)),
			"expected hex string of length %d, but got `%s`",
			hex.EncodedLen(len(h)),
			b,
		)
	}
	_, err := hex.Decode(h[:], b)
	return err
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
// If b is a JSON-encoded null, it copies the zero-value into h. Othwerwise, it
// decodes hex data from b into h.
func (h *Hash) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		*h = Hash{}
		return nil
	}

	s := new(string)
	err := json.Unmarshal(b, s)
	if err != nil {
		return err
	}

	return h.UnmarshalText([]byte(*s))
}

// Value satisfies the driver.Valuer interface
func (h Hash) Value() (driver.Value, error) {
	return h[:], nil
}

// Scan satisfies the driver.Scanner interface
func (h *Hash) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		copy(h[:], v)
		return nil
	default:
		return fmt.Errorf("Hash.Scan received unsupported type %T", val)
	}
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

func WriteFastHash(w io.Writer, d []byte) error {
	if len(d) == 0 {
		_, err := blockchain.WriteVarstr31(w, nil)
		return err
	}
	var h [32]byte
	sha3pool.Sum256(h[:], d)
	_, err := blockchain.WriteVarstr31(w, h[:])
	return err
}
