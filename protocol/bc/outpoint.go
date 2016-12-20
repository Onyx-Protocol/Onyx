package bc

import (
	"io"
	"strconv"

	"chain/encoding/blockchain"
)

// Outpoint defines a bitcoin data type that is used to track previous
// transaction outputs.
type Outpoint struct {
	Hash  Hash   `json:"hash"`
	Index uint32 `json:"index"`
}

// WriteTo writes p to w.
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

// String returns the Outpoint in the human-readable form "hash:index".
func (p Outpoint) String() string {
	return p.Hash.String() + ":" + strconv.FormatUint(uint64(p.Index), 10)
}
