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

var EmptyStringHash *Hash

func init() {
	EmptyStringHash = new(Hash)
	EmptyStringHash.FromByte32(sha3.Sum256(nil))
}

// Hash represents a 256-bit hash.  Data structure defined in
// hash.proto.

func (h *Hash) Byte32() (b32 [32]byte) {
	binary.BigEndian.PutUint64(b32[0:8], h.V0)
	binary.BigEndian.PutUint64(b32[8:16], h.V1)
	binary.BigEndian.PutUint64(b32[16:24], h.V2)
	binary.BigEndian.PutUint64(b32[24:32], h.V3)
	return b32
}

func (h *Hash) FromByte32(b32 [32]byte) {
	h.V0 = binary.BigEndian.Uint64(b32[0:8])
	h.V1 = binary.BigEndian.Uint64(b32[8:16])
	h.V2 = binary.BigEndian.Uint64(b32[16:24])
	h.V3 = binary.BigEndian.Uint64(b32[24:32])
}

// String returns the bytes of h encoded in hex.
func (h *Hash) String() string {
	b, _ := h.MarshalText() // #nosec
	return string(b)
}

// MarshalText satisfies the TextMarshaler interface.
// It returns the bytes of h encoded in hex,
// for formats that can't hold arbitrary binary data.
// It never returns an error.
func (h *Hash) MarshalText() ([]byte, error) {
	b32 := h.Byte32()
	b := make([]byte, hex.EncodedLen(len(b32)))
	hex.Encode(b, b32[:])
	return b, nil
}

// UnmarshalText satisfies the TextUnmarshaler interface.
// It decodes hex data from b into h.
func (h *Hash) UnmarshalText(b []byte) error {
	var b32 [32]byte
	if len(b) != hex.EncodedLen(len(b32)) {
		return errors.WithDetailf(
			fmt.Errorf("bad hash hex length %d", len(b)),
			"expected hex string of length %d, but got `%s`",
			hex.EncodedLen(len(b32)),
			b,
		)
	}
	_, err := hex.Decode(b32[:], b)
	if err != nil {
		return err
	}
	h.FromByte32(b32)
	return nil
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

func (h Hash) Bytes() []byte {
	b32 := h.Byte32()
	return b32[:]
}

// Value satisfies the driver.Valuer interface
func (h Hash) Value() (driver.Value, error) {
	return h.Bytes(), nil
}

// Scan satisfies the driver.Scanner interface
func (h *Hash) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		var b32 [32]byte
		copy(b32[:], v)
		h.FromByte32(b32)
		return nil
	default:
		return fmt.Errorf("Hash.Scan received unsupported type %T", val)
	}
}

// ParseHash takes a hex-encoded hash and returns
// a 32 byte array.
func ParseHash(s string) (h *Hash, err error) {
	var b32 [32]byte
	if len(s) != hex.EncodedLen(len(b32)) {
		return h, errors.New("wrong hex length")
	}
	_, err = hex.Decode(b32[:], []byte(s))
	h = new(Hash)
	h.FromByte32(b32)
	return h, errors.Wrap(err, "decode hex")
}

func writeFastHash(w io.Writer, d []byte) error {
	if len(d) == 0 {
		_, err := blockchain.WriteVarstr31(w, nil)
		return err
	}
	var h [32]byte
	sha3pool.Sum256(h[:], d)
	_, err := blockchain.WriteVarstr31(w, h[:])
	return err
}

// WriteTo writes p to w.
func (h *Hash) WriteTo(w io.Writer) (int64, error) {
	b32 := h.Byte32()
	n, err := w.Write(b32[:])
	return int64(n), err
}

func (h *Hash) readFrom(r io.Reader) (int, error) {
	var b32 [32]byte
	n, err := io.ReadFull(r, b32[:])
	if err != nil {
		return n, err
	}
	h.FromByte32(b32)
	return n, nil
}
