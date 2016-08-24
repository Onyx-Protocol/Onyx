package state

import "chain/protocol/bc"

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
