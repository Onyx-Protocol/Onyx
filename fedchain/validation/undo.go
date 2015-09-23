package validation

import (
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

// UndoBlock holds information necessary to undo the effects
// of applying a block. Applying an UndoBlock will restore the
// state that existed before the block was applied.
type UndoBlock struct {
	// Undos for the transactions in the same order as transactions
	// in a block, therefore they must be applied in reverse order.
	UndoTxs []*UndoTx
}

// UndoTx holds information necessary to undo the effects
// of applying a tx. Applying an UndoTx will restore the
// state that existed before the tx was applied.
type UndoTx struct {
	// Undos for the inputs in the same order as inputs in a transaction,
	// therefore they must be applied in reverse order.
	UndoInputs []*UndoTxInput
}

// UndoTxInput holds information necessary to undo the effects
// of applying an input. Applying an UndoTxInput will restore the
// state that existed before the input was applied.
type UndoTxInput struct {
	Output state.Output
	ADP    *bc.AssetDefinitionPointer // may be nil
}
