package bc

import (
	"database/sql/driver"
	"io"
	"strconv"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
)

// Outpoint is a raw txhash+index pointer to an output.
type Outpoint struct {
	Hash  Hash   `json:"hash"`
	Index uint32 `json:"index"`
}

// String returns the Outpoint in the human-readable form "hash:index".
func (p Outpoint) String() string {
	return p.Hash.String() + ":" + strconv.FormatUint(uint64(p.Index), 10)
}

// WriteTo writes p to w.
// It assumes w has sticky errors.
func (p *Outpoint) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(p.Hash[:])
	if err != nil {
		return int64(n), err
	}
	n2, err := blockchain.WriteVarint31(w, uint64(p.Index))
	return int64(n + n2), err
}

func (p *Outpoint) readFrom(r io.Reader) (int, error) {
	n1, err := io.ReadFull(r, p.Hash[:])
	if err != nil {
		return n1, err
	}
	var n2 int
	p.Index, n2, err = blockchain.ReadVarint31(r)
	return n1 + n2, err
}

// OutputID identifies previous transaction output in transaction inputs.
type OutputID Hash

func (o OutputID) String() string                { return Hash(o).String() }
func (o OutputID) MarshalText() ([]byte, error)  { return Hash(o).MarshalText() }
func (o *OutputID) UnmarshalText(b []byte) error { return (*Hash)(o).UnmarshalText(b) }
func (o *OutputID) UnmarshalJSON(b []byte) error { return (*Hash)(o).UnmarshalJSON(b) }
func (o OutputID) Value() (driver.Value, error)  { return Hash(o).Value() }
func (o *OutputID) Scan(b interface{}) error     { return (*Hash)(o).Scan(b) }

// ComputeOutpoint computes the outpoint defined by transaction hash, output index and output hash.
func ComputeOutputID(txHash Hash, outputIndex uint32, outputHash Hash) (outputid OutputID) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	h.Write(txHash[:])
	blockchain.WriteVarint31(h, uint64(outputIndex))
	h.Write(outputHash[:])
	h.Read(outputid[:])
	return outputid
}

// WriteTo writes p to w.
// It assumes w has sticky errors.
func (outid *OutputID) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(outid[:])
	return int64(n), err
}

func (outid *OutputID) readFrom(r io.Reader) (int, error) {
	return io.ReadFull(r, outid[:])
}


