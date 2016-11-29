package state

import (
	"bytes"

	"chain/errors"
	"chain/protocol/bc"
)

// Output represents a spent or unspent output
// for the validation process.
type Output struct {
	bc.Outpoint
	bc.TxOutput
}

// NewOutput creates a new Output.
func NewOutput(o bc.TxOutput, p bc.Outpoint) *Output {
	return &Output{
		TxOutput: o,
		Outpoint: p,
	}
}

// Prevout returns the Output consumed by the provided tx input. It
// only includes the output data that is embedded within inputs (ex,
// excludes reference data).
func Prevout(in *bc.TxInput) *Output {
	assetAmount := in.AssetAmount()
	t := bc.NewTxOutput(assetAmount.AssetID, assetAmount.Amount, in.ControlProgram(), nil)
	return &Output{
		Outpoint: in.Outpoint(),
		TxOutput: *t,
	}
}

// OutputKey returns the key of an output in the state tree.
func OutputKey(o bc.Outpoint) (bkey []byte) {
	var b bytes.Buffer
	w := errors.NewWriter(&b) // used to satisfy interfaces
	o.WriteTo(w)
	return b.Bytes()
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
	return OutputKey(o.Outpoint), outputBytes(o)
}
