package state

import (
	"bytes"

	"chain/protocol/bc"
)

// Output represents a spent or unspent output
// for the validation process.
type Output struct {
	bc.OutputID
	bc.TxOutput
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
	t := bc.NewTxOutput(assetAmount.AssetID, assetAmount.Amount, in.ControlProgram(), nil)
	return &Output{
		OutputID: in.OutputID(),
		TxOutput: *t,
	}
}

// OutputKey returns the key of an output in the state tree.
func OutputKey(o bc.UnspentID) (bkey []byte) {
	// TODO(oleg): check if we no longer need this buffer writing.
	return o[:]
	// var b bytes.Buffer
	// w := errors.NewWriter(&b) // used to satisfy interfaces
	// o.WriteTo(w)
	// return b.Bytes()
}

func outputBytes(o *Output) []byte {
	var b bytes.Buffer
	o.WriteCommitment(&b)
	return b.Bytes()
}

// OutputTreeItem returns the key of an output in the state tree,
// as well as the output commitment (a second []byte) for Inserts
// into the state tree.
func OutputTreeItem(o *Output) (bkey, commitment []byte) {
	// TODO(oleg): replace value with the key, so we can later optimize the tree to become a set.
	key := OutputKey(bc.ComputeUnspentID(o.OutputID, o.TxOutput.CommitmentHash()))
	return key, outputBytes(o)
}
