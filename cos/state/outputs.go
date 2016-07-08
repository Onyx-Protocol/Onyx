package state

import (
	"bytes"

	"chain/cos/bc"
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
// excludes metadata).
func Prevout(in *bc.TxInput) *Output {
	return &Output{
		Outpoint: in.Previous,
		TxOutput: bc.TxOutput{
			AssetAmount: in.AssetAmount,
			Script:      in.PrevScript,
		},
	}
}

// OutputSet identifies a set of transaction outputs.
type OutputSet struct {
	outputs map[bc.Outpoint]*Output
}

// Contains returns true iff the provided Output exists in the set. Outputs
// are compared by outpoint, asset amount and control program.
func (os OutputSet) Contains(o *Output) bool {
	output, ok := os.outputs[o.Outpoint]
	if !ok {
		return false
	}
	return output.AssetAmount == o.AssetAmount && bytes.Equal(output.Script, o.Script)
}

// NewOutputSet constructs a new OutputSet from the provided
// outputs.
func NewOutputSet(outputs ...*Output) OutputSet {
	if len(outputs) == 0 {
		return OutputSet{}
	}

	m := make(map[bc.Outpoint]*Output, len(outputs))
	for _, o := range outputs {
		m[o.Outpoint] = o
	}
	return OutputSet{outputs: m}
}
