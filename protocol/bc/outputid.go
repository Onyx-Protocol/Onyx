package bc

import (
	"database/sql/driver"
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
)

// OutputID identifies previous transaction output in transaction inputs.
type OutputID Hash

// UnspentID identifies and commits to unspent output.
type UnspentID Hash

func (o OutputID) Bytes() []byte                 { return Hash(o).Bytes() }
func (o OutputID) String() string                { return Hash(o).String() }
func (o OutputID) MarshalText() ([]byte, error)  { return Hash(o).MarshalText() }
func (o *OutputID) UnmarshalText(b []byte) error { return (*Hash)(o).UnmarshalText(b) }
func (o *OutputID) UnmarshalJSON(b []byte) error { return (*Hash)(o).UnmarshalJSON(b) }
func (o OutputID) Value() (driver.Value, error)  { return Hash(o).Value() }
func (o *OutputID) Scan(b interface{}) error     { return (*Hash)(o).Scan(b) }

// ComputeOutputID computes the output ID defined by transaction hash and output index.
func ComputeOutputID(txHash Hash, outputIndex uint32) (oid OutputID) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	h.Write(txHash[:])
	blockchain.WriteVarint31(h, uint64(outputIndex))
	h.Read(oid[:])
	return oid
}

func (u UnspentID) Bytes() []byte                 { return Hash(u).Bytes() }
func (u UnspentID) String() string                { return Hash(u).String() }
func (u UnspentID) MarshalText() ([]byte, error)  { return Hash(u).MarshalText() }
func (u *UnspentID) UnmarshalText(b []byte) error { return (*Hash)(u).UnmarshalText(b) }
func (u *UnspentID) UnmarshalJSON(b []byte) error { return (*Hash)(u).UnmarshalJSON(b) }
func (u UnspentID) Value() (driver.Value, error)  { return Hash(u).Value() }
func (u *UnspentID) Scan(b interface{}) error     { return (*Hash)(u).Scan(b) }

// ComputeOutputID computes the output ID defined by transaction hash, output index and output hash.
func ComputeUnspentID(oid OutputID, outputHash Hash) (uid UnspentID) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	h.Write(oid[:])
	h.Write(outputHash[:])
	h.Read(uid[:])
	return uid
}

// WriteTo writes p to w.
func (outid *OutputID) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(outid[:])
	return int64(n), err
}

func (outid *OutputID) readFrom(r io.Reader) (int, error) {
	return io.ReadFull(r, outid[:])
}
