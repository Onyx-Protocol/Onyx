package state

import (
	"chain/protocol/bc"
)

// Output represents a spent or unspent output
// for the validation process.
type Output struct {
	bc.OutputID
}

// NewOutput creates a new Output.
func NewOutput(outid bc.OutputID) *Output {
	return &Output{
		OutputID: outid,
	}
}

// Prevout returns the Output consumed by the provided tx input. It
// only includes the output data that is embedded within inputs (ex,
// excludes reference data).
func Prevout(in *bc.TxInput) *Output {
	return &Output{
		OutputID: in.SpentOutputID(),
	}
}

// OutputTreeItem returns the key of an output in the state tree,
// as well as the output commitment (a second []byte) for Inserts
// into the state tree.
func OutputTreeItem(o *Output) (bkey, commitment []byte) {
	// We implement the set of unspent IDs via Patricia Trie
	// by having the leaf data being equal to keys.
	key := o.OutputID.Bytes()
	return key, key
}
