package validation

import (
	"errors"

	"chain/fedchain/bc"
	"chain/fedchain/state"
)

// ValidateTx validates the given transaction
// against the given state and applies its
// changes to the view.
func ValidateTx(view state.View, tx *bc.Tx) error {
	// TODO: Check that transactions are valid.
	for _, in := range tx.Inputs {
		o := view.Output(in.Previous)
		if o == nil || o.Spent {
			return errors.New("previous output is spent or missing")
		}

		o.Spent = true
		view.SaveOutput(o)
	}

	for _, out := range tx.Outputs {
		o := &state.Output{
			TxOutput: *out,
			Outpoint: out.Outpoint,
			Spent:    false,
		}
		view.SaveOutput(o)
	}
	return nil
}
