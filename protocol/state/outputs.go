package state

import (
	"chain/protocol/bc"
)

// Output represents a spent or unspent output
// for the validation process.
type Output struct {
	bc.OutputID
	bc.TxOutput
}

func (o *Output) UnspentID() bc.UnspentID {
	return bc.ComputeUnspentID(o.OutputID, o.TxOutput.CommitmentHash())
}

// NewOutput creates a new Output.
func NewOutput(o bc.TxOutput, outid bc.OutputID) *Output {
	return &Output{
		TxOutput: o,
		OutputID: outid,
	}
}

// Prevout returns the Output consumed by the provided tx input. It
// only includes the output data that is embedded within inputs (ex,
// excludes reference data).
func Prevout(in *bc.TxInput) *Output {
	assetAmount := in.AssetAmount()
	// TODO(oleg): for new outputid we need to have correct output commitment, not reconstruct this here
	// Also we do not care about all these, but only about UnspentID
	t := bc.NewTxOutput(assetAmount.AssetID, assetAmount.Amount, in.ControlProgram(), nil)
	return &Output{
		OutputID: in.SpentOutputID(),
		TxOutput: *t,
	}
}

// OutputTreeItem returns the key of an output in the state tree,
// as well as the output commitment (a second []byte) for Inserts
// into the state tree.
func OutputTreeItem(o *Output) (bkey, commitment []byte) {
	// We implement the set of unspent IDs via Patricia Trie
	// by having the leaf data being equal to keys.
	key := o.UnspentID().Bytes()
	return key, key
}
