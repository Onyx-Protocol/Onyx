package bc

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"chain/errors"
)

type Byte32 [32]byte

// String returns the bytes of b32 encoded in hex.
func (b32 Byte32) String() string {
	b, _ := b32.MarshalText()
	return string(b)
}

// MarshalText satisfies the TextMarshaler interface.
// It returns the bytes of b32 encoded in hex,
// for formats that can't hold arbitrary binary data.
// It never returns an error.
func (b32 Byte32) MarshalText() ([]byte, error) {
	b := make([]byte, hex.EncodedLen(len(b32)))
	hex.Encode(b, b32[:])
	return b, nil
}

// UnmarshalText satisfies the TextUnmarshaler interface.
// It decodes hex data from b into b32.
func (b32 *Byte32) UnmarshalText(b []byte) error {
	if len(b) != hex.EncodedLen(len(*b32)) {
		return errors.WithDetailf(
			fmt.Errorf("bad hash hex length %d", len(b)),
			"expected hex string of length %d, but got `%s`",
			hex.EncodedLen(len(*b32)),
			b,
		)
	}
	_, err := hex.Decode(b32[:], b)
	return err
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
// If b is a JSON-encoded null, it copies the zero-value into h. Othwerwise, it
// decodes hex data from b into h.
func (b32 *Byte32) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		*b32 = Byte32{}
		return nil
	}
	s := new(string)
	err := json.Unmarshal(b, s)
	if err != nil {
		return err
	}
	return b32.UnmarshalText([]byte(*s))
}

// Value satisfies the driver.Valuer interface.
func (b32 Byte32) Value() (driver.Value, error) {
	return b32[:], nil
}

// Scan satisfies the driver.Scanner interface.
func (b32 *Byte32) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		copy(b32[:], v)
		return nil
	default:
		return fmt.Errorf("Hash.Scan received unsupported type %T", val)
	}
}

// WriteTo satisfies the io.WriterTo interface.
func (b32 Byte32) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(b32[:])
	return int64(n), err
}

// ReadFrom satisfies the io.ReaderFrom interface.
func (b32 *Byte32) ReadFrom(r io.Reader) (int64, error) {
	n, err := io.ReadFull(r, b32[:])
	return int64(n), err
}
