package bc

import (
//	"database/sql/driver"
)

// Outpoint is a raw txhash+index pointer to an output.
type Outpoint struct {
	Hash  Hash   `json:"hash"`
	Index uint32 `json:"index"`
}

// OutputID identifies previous transaction output in transaction inputs.
type OutputID struct {
	Hash  Hash   `json:"hash"`
	Index uint32 `json:"index"`
}

// TODO(oleg): replace above struct with `type OutputID Hash`.

// TODO(oleg): rewrite these in terms of existing struct so we can refactor all other code to use this API and not break tests.
// func (o OutputID) String() string                { return Hash(o).String() }
// func (o OutputID) MarshalText() ([]byte, error)  { return Hash(o).MarshalText() }
// func (o *OutputID) UnmarshalText(b []byte) error { return (*Hash)(o).UnmarshalText(b) }
// func (o *OutputID) UnmarshalJSON(b []byte) error { return (*Hash)(o).UnmarshalJSON(b) }
// func (o OutputID) Value() (driver.Value, error)  { return Hash(o).Value() }
// func (o *OutputID) Scan(b interface{}) error     { return (*Hash)(o).Scan(b) }

// ComputeOutpoint computes the outpoint defined by transaction hash, output index and output hash.
// TODO(oleg): add `, outputHash Hash` argument to this function.
func ComputeOutputID(txHash Hash, outputIndex uint32) (outputid OutputID) {
	// TODO(oleg): rewrite into sha3(txhash || uint64le(index) || outputhash)
	return OutputID{
		Hash:  txHash,
		Index: outputIndex,
	}
}
