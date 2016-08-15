package validation

import (
	"bytes"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/cos/txscript"
	"chain/errors"
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
	ErrBadStateRoot = errors.New("invalid state merkle root")
)

// ValidateAndApplyBlock validates the given block against the given
// state tree and applies its changes to the state snapshot.
// If block is invalid, it returns a non-nil error describing why.
func ValidateAndApplyBlock(ctx context.Context, snapshot *state.Snapshot, prevBlock, block *bc.Block) error {
	return validateBlock(ctx, snapshot, prevBlock, block, true)
}

// ValidateBlockForSig performs validation on an incoming _unsigned_
// block in preparation for signing it.  By definition it does not
// execute the sigscript.
func ValidateBlockForSig(ctx context.Context, snapshot *state.Snapshot, prevBlock, block *bc.Block) error {
	return validateBlock(ctx, snapshot, prevBlock, block, false)
}

func validateBlock(ctx context.Context, snapshot *state.Snapshot, prevBlock, block *bc.Block, runScript bool) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	err := validateBlockHeader(prevBlock, block, runScript)
	if err != nil {
		return err
	}

	// TODO: Check that other block headers are valid.
	// TODO(erywalder): consider writing to a copy of the state tree
	// of the one provided and make the caller call ApplyBlock as well
	for _, tx := range block.Transactions {
		// TODO(jackson): This ValidateTx call won't be necessary if this
		// tx is in the pool. It'll be cleaner to implement once prevout
		// commitments are up to spec.
		err := ValidateTx(tx)
		if err != nil {
			return err
		}
		err = ConfirmTx(snapshot, tx, block.TimestampMS)
		if err != nil {
			return err
		}
		err = ApplyTx(snapshot, tx)
		if err != nil {
			return err
		}
	}

	if block.StateRoot() != snapshot.Tree.RootHash() {
		return ErrBadStateRoot
	}
	return nil
}

// ApplyBlock applies the transactions in the block to the state tree.
func ApplyBlock(snapshot *state.Snapshot, block *bc.Block) error {
	for _, tx := range block.Transactions {
		err := ApplyTx(snapshot, tx)
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
func ValidateBlockHeader(prevBlock, block *bc.Block) error {
	return validateBlockHeader(prevBlock, block, true)
}

func validateBlockHeader(prevBlock, block *bc.Block, runScript bool) error {
	if prevBlock == nil && block.Height != 1 {
		return ErrBadHeight
	}
	if prevBlock != nil {
		prevHash := prevBlock.Hash()
		if !bytes.Equal(block.PreviousBlockHash[:], prevHash[:]) {
			return ErrBadPrevHash
		}
		if block.Height != prevBlock.Height+1 {
			return ErrBadHeight
		}
		if block.TimestampMS < prevBlock.TimestampMS {
			return ErrBadTimestamp
		}
	}

	txMerkleRoot := CalcMerkleRoot(block.Transactions)
	// can be modified to allow soft fork
	if block.TxRoot() != txMerkleRoot {
		return ErrBadTxRoot
	}

	if txscript.IsUnspendable(block.OutputScript) {
		return ErrBadScript
	}

	if runScript && prevBlock != nil {
		engine, err := txscript.NewEngineForBlock(prevBlock.OutputScript, block, txscript.StandardVerifyFlags)
		if err != nil {
			return err
		}
		if err = engine.Execute(); err != nil {
			pkScriptStr, _ := txscript.DisasmString(prevBlock.OutputScript)
			sigScriptStr, _ := txscript.DisasmString(block.SignatureScript)
			return errors.Wrapf(ErrBadSig, "validation failed in script execution in block (sigscript[%s] pkscript[%s]): %s", sigScriptStr, pkScriptStr, err.Error())
		}
	}

	return nil
}
