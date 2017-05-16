package bc

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/sha3"

	"chain/errors"
	"chain/protocol/bc/internal"
)

// Hash represents a 256-bit hash.
type Hash [32]byte

var EmptyStringHash Hash = sha3.Sum256(nil)

func (m Hash) Marshal() ([]byte, error) {
	var h internal.Hash
	h.V0 = binary.BigEndian.Uint64(m[0:8])
	h.V1 = binary.BigEndian.Uint64(m[8:16])
	h.V2 = binary.BigEndian.Uint64(m[16:24])
	h.V3 = binary.BigEndian.Uint64(m[24:32])
	return proto.Marshal(&h)
}

func (m *Hash) Unmarshal(b []byte) error {
	var h internal.Hash
	err := proto.Unmarshal(b, &h)
	if err != nil {
		return errors.Wrap(err)
	}
	binary.BigEndian.PutUint64(m[0:8], h.V0)
	binary.BigEndian.PutUint64(m[8:16], h.V1)
	binary.BigEndian.PutUint64(m[16:24], h.V2)
	binary.BigEndian.PutUint64(m[24:32], h.V3)
	return nil
}

func NewHash(b32 [32]byte) (h Hash) {
	return Hash(b32)
}

func (h Hash) Byte32() (b32 [32]byte) {
	return [32]byte(h)
}

// MarshalText satisfies the TextMarshaler interface.
// It returns the bytes of h encoded in hex,
// for formats that can't hold arbitrary binary data.
// It never returns an error.
func (h Hash) MarshalText() ([]byte, error) {
	b := h.Byte32()
	v := make([]byte, 64)
	hex.Encode(v, b[:])
	return v, nil
}

// UnmarshalText satisfies the TextUnmarshaler interface.
// It decodes hex data from b into h.
func (h *Hash) UnmarshalText(v []byte) error {
	var b [32]byte
	if len(v) != 64 {
		return fmt.Errorf("bad length hash string %d", len(v))
	}
	_, err := hex.Decode(b[:], v)
	*h = NewHash(b)
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
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	return h.UnmarshalText([]byte(s))
}

func (h Hash) Bytes() []byte {
	b32 := h.Byte32()
	return b32[:]
}

// Value satisfies the driver.Valuer interface
func (h Hash) Value() (driver.Value, error) {
	return h.Bytes(), nil
}

// Scan satisfies the driver.Scanner interface
func (h *Hash) Scan(v interface{}) error {
	var buf [32]byte
	b, ok := v.([]byte)
	if !ok {
		return fmt.Errorf("Hash.Scan received unsupported type %T", v)
	}
	copy(buf[:], b)
	*h = NewHash(buf)
	return nil
}

// WriteTo satisfies the io.WriterTo interface.
func (h Hash) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(h.Bytes())
	return int64(n), err
}

// ReadFrom satisfies the io.ReaderFrom interface.
func (h *Hash) ReadFrom(r io.Reader) (int64, error) {
	var b32 [32]byte
	n, err := io.ReadFull(r, b32[:])
	if err != nil {
		return int64(n), err
	}
	*h = NewHash(b32)
	return int64(n), nil
}

// IsZero tells whether a Hash pointer is nil or points to an all-zero
// hash.
func (h *Hash) IsZero() bool {
	if h == nil {
		return true
	}
	return *h == Hash{}
}
