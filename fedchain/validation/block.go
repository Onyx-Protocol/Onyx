package validation

import (
	"bytes"

	"golang.org/x/net/context"

	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
	"chain/net/trace/span"
)

// Errors returned by ValidateAndApplyBlock
var (
	ErrBadPrevHash  = errors.New("invalid previous block hash")
	ErrBadHeight    = errors.New("invalid block height")
	ErrBadTimestamp = errors.New("invalid block timestamp")
	ErrBadScript    = errors.New("unspendable block script")
	ErrBadSig       = errors.New("invalid signature script")
	ErrBadTxRoot    = errors.New("invalid transaction merkle root")
)

// ValidateAndApplyBlock validates the given block
// against the given state and applies its
// changes to the view.
// If block is invalid,
// it returns a non-nil error describing why.
func ValidateAndApplyBlock(ctx context.Context, view state.View, prevBlock, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	err := ValidateBlockHeader(ctx, prevBlock, block)
	if err != nil {
		return err
	}

	// TODO: Check that other block headers are valid.
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

// ValidateBlockHeader validates all pieces of the header
// that can be checked before processing the transactions.
// This includes the previous block hash, height, timestamp,
// output script, and signature script.
func ValidateBlockHeader(ctx context.Context, prevBlock, block *bc.Block) error {
	prevHash := prevBlock.Hash()
	if !bytes.Equal(block.PreviousBlockHash[:], prevHash[:]) {
		return ErrBadPrevHash
	}

	txMerkleRoot := CalcMerkleRoot(block.Transactions)
	// can be modified to allow soft fork
	if !bytes.Equal(block.TxRoot[:], txMerkleRoot[:]) {
		return ErrBadTxRoot
	}

	if block.Height != prevBlock.Height+1 {
		return ErrBadHeight
	}

	if block.Timestamp < prevBlock.Timestamp {
		return ErrBadTimestamp
	}

	if txscript.IsUnspendable(block.OutputScript) {
		return ErrBadScript
	}

	engine, err := txscript.NewEngineForBlock(ctx, prevBlock.OutputScript, block, txscript.StandardVerifyFlags)
	if err != nil {
		return err
	}
	if err = engine.Execute(); err != nil {
		pkScriptStr, _ := txscript.DisasmString(prevBlock.OutputScript)
		sigScriptStr, _ := txscript.DisasmString(block.SignatureScript)
		return errors.Wrapf(ErrBadSig, "validation failed in script execution in block (sigscript[%s] pkscript[%s]): %s", sigScriptStr, pkScriptStr, err.Error())
	}
	return nil
}
