package validation

import (
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

// ValidateBlock validates the given block
// against the given state and applies its
// changes to the view.
func ValidateBlock(view state.View, block *bc.Block) error {
	// TODO: Check that block headers are valid.
	for _, tx := range block.Transactions {
		err := ValidateTx(view, tx)
		if err != nil {
			return err
		}
	}
	return nil
}
