package state

import (
	"bytes"

	"chain-stealth/errors"
	"chain-stealth/protocol/bc"
)

// Output represents a spent or unspent output
// for the validation process.
type Output struct {
	bc.Outpoint
	bc.TypedOutput
}

// NewOutput creates a new Output.
func NewOutput(o bc.TypedOutput, p bc.Outpoint) *Output {
	return &Output{
		TypedOutput: o,
		Outpoint:    p,
	}
}

// Prevout returns the Output consumed by the provided tx input, which
// should be a spend. It only includes the output data that is
// embedded within inputs (ex, excludes reference data).
func Prevout(in *bc.TxInput) *Output {
	sp, ok := in.TypedInput.(*bc.SpendInput)
	if !ok {
		return nil
	}
	return NewOutput(sp.TypedOutput, sp.Outpoint)
}

// OutputKey returns the key of an output in the state tree.
func OutputKey(o bc.Outpoint) (bkey []byte) {
	var b bytes.Buffer
	w := errors.NewWriter(&b) // used to satisfy interfaces
	o.WriteTo(w)
	return b.Bytes()
}

func OutputBytes(o *Output) []byte {
	var b bytes.Buffer
	o.TypedOutput.WriteTo(&b)
	return b.Bytes()
}

// OutputTreeItem returns the key of an output in the state tree,
// as well as the output commitment (a second []byte) for Inserts
// into the state tree.
func OutputTreeItem(o *Output) (bkey, commitment []byte) {
	return OutputKey(o.Outpoint), OutputBytes(o)
}
