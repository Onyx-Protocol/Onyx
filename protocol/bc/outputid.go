package bc

import (
    "database/sql/driver"

    "chain/crypto/sha3pool"
    "chain/encoding/blockchain"
)

// Outpoint is a raw txhash+index pointer to an output.
type Outpoint struct {
	Hash  Hash   `json:"hash"`
	Index uint32 `json:"index"`
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
