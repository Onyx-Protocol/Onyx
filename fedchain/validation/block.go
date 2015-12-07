package validation

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/fedchain/state"
)

// ValidateBlock validates the given block
// against the given state and applies its
// changes to the view.
// If block is invalid,
// it returns a non-nil error describing why.
func ValidateBlock(ctx context.Context, view state.View, block *bc.Block) error {
	// TODO: Check that block headers are valid.
	// TODO(kr): consider splitting ApplyTx out of ValidateTx
	// and therefore making the user call ApplyBlock separately.
	for _, tx := range block.Transactions {
		err := ValidateTx(ctx, view, tx, block.Timestamp, &block.PreviousBlockHash)
		if err != nil {
			return err
		}
	}
	return nil
}

func ApplyBlock(ctx context.Context, view state.View, block *bc.Block) error {
	for _, tx := range block.Transactions {
		err := ApplyTx(ctx, view, tx)
		if err != nil {
			return err
		}
	}
	return nil
}
