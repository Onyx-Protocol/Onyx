package bc

import (
//	"database/sql/driver"
)

// Outpoint identifies previous transaction output.
type Outpoint struct {
	Hash  Hash   `json:"hash"`
	Index uint32 `json:"index"`
}

// TODO(oleg): type Outpoint Hash

// TODO(oleg): rewrite these in terms of existing struct so we can refactor all other code to use this API and not break tests.
// func (o Outpoint) String() string                { return Hash(o).String() }
// func (o Outpoint) MarshalText() ([]byte, error)  { return Hash(o).MarshalText() }
// func (o *Outpoint) UnmarshalText(b []byte) error { return (*Hash)(o).UnmarshalText(b) }
// func (o *Outpoint) UnmarshalJSON(b []byte) error { return (*Hash)(o).UnmarshalJSON(b) }
// func (o Outpoint) Value() (driver.Value, error)  { return Hash(o).Value() }
// func (o *Outpoint) Scan(b interface{}) error     { return (*Hash)(o).Scan(b) }

// ComputeOutpoint computes the outpoint defined by transaction hash, output index and output hash.
func ComputeOutpoint(txHash Hash, outputIndex uint32, outputHash Hash) (outpoint Outpoint) {
	// TODO(oleg): rewrite into sha3(txhash || uint64le(index) || outhash)
	return Outpoint{
		Hash: txHash,
		Index: outputIndex,
	}
}


