package validation

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/net/trace/span"
)

// ValidateAndApplyBlock validates the given block
// against the given state and applies its
// changes to the view.
// If block is invalid,
// it returns a non-nil error describing why.
func ValidateAndApplyBlock(ctx context.Context, view state.View, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	// TODO: Check that block headers are valid.
	// TODO(erywalder): consider writing to a temporary view instead
	// of the one provided and make the caller call ApplyBlock as well
	for _, tx := range block.Transactions {
		err := ValidateTx(ctx, view, tx, block.Timestamp, &block.PreviousBlockHash)
		if err != nil {
			return err
		}
		err = ApplyTx(ctx, view, tx)
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
