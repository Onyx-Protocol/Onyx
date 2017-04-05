package bc

import (
	"database/sql/driver"
	"encoding/binary"
	"io"

	"golang.org/x/crypto/sha3"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
)

// Hash represents a 256-bit hash.

var EmptyStringHash = NewHash(sha3.Sum256(nil))

func NewHash(b [32]byte) (h Hash) {
	h.FromByte32(b)
	return h
}

func (h Hash) Byte32() (b32 [32]byte) {
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

// MarshalText satisfies the TextMarshaler interface.
// It returns the bytes of h encoded in hex,
// for formats that can't hold arbitrary binary data.
// It never returns an error.
func (h Hash) MarshalText() ([]byte, error) {
	return byte32(h.Byte32()).MarshalText()
}

// UnmarshalText satisfies the TextUnmarshaler interface.
// It decodes hex data from b into h.
func (h *Hash) UnmarshalText(b []byte) error {
	var b32 byte32
	err := b32.UnmarshalText(b)
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
	var b32 byte32
	err := b32.UnmarshalJSON(b)
	if err != nil {
		return err
	}
	h.FromByte32(b32)
	return nil
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
	var b32 byte32
	err := b32.Scan(val)
	if err != nil {
		return err
	}
	h.FromByte32(b32)
	return nil
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

// WriteTo satisfies the io.WriterTo interface.
func (h *Hash) WriteTo(w io.Writer) (int64, error) {
	return byte32(h.Byte32()).WriteTo(w)
}

// WriteTo satisfies the io.ReaderFrom interface.
func (h *Hash) ReadFrom(r io.Reader) (int64, error) {
	var b32 byte32
	n, err := b32.ReadFrom(r)
	if err != nil {
		return n, err
	}
	h.FromByte32(b32)
	return n, nil
}
