package validation

import (
	"fmt"
	"os"

	"golang.org/x/net/context"

	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
)

var stubGenesisHash = bc.Hash{}

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// ValidateTxInputs just validates that the tx inputs are present
// and unspent in the view.
func ValidateTxInputs(ctx context.Context, view state.ViewReader, tx *bc.Tx) error {
	// Verify inputs for double-spends and update ADPs on the view.
	for inIndex, txin := range tx.Inputs {
		if txin.IsIssuance() {
			continue
		}
		unspent := view.Output(ctx, txin.Previous)
		// It's possible to load a spent output here because BackedView
		// explicitly stores spent outputs in frontend to shadow unspent
		// outputs in backend.
		if unspent == nil || unspent.Spent {
			return errors.WithDetailf(ErrBadTx, "output %s for input %d is invalid or already spent", txin.Previous.String(), inIndex)
		}
	}
	return nil
}

// ValidateTx validates the given transaction
// against the given state and applies its
// changes to the view.
// If tx is invalid,
// it returns a non-nil error describing why.
func ValidateTx(ctx context.Context, view state.ViewReader, tx *bc.Tx, timestamp uint64) error {
	// Don't make a span here, because there are too many of them
	// to comfortably fit in a single trace for processing (creating
	// or applying) a block.
	// TODO(kr): ask Ben what's reasonable to do in this situation.

	err := txIsWellFormed(tx)
	if err != nil {
		return errors.Wrap(err, "well-formed test")
	}

	// Check time
	if tx.LockTime > timestamp {
		return errors.WithDetail(ErrBadTx, "transaction lock time is in the future")
	}

	// If this is an issuance tx, check its prevout hash against the
	// previous block hash if we have one.
	// NOTE: review this when we implement import inputs.
	// Maybe we'll need to have undo ADP.
	// TODO(erykwalder): some type of uniqueness check

	err = ValidateTxInputs(ctx, view, tx)
	if err != nil {
		return errors.Wrap(err, "validating inputs")
	}

	err = validateTxBalance(ctx, view, tx)
	if err != nil {
		return errors.Wrap(err, "validating balance")
	}

	engine, err := txscript.NewReusableEngine(ctx, view, &tx.TxData, txscript.StandardVerifyFlags)
	if err != nil {
		return fmt.Errorf("cannot create script engine: %s", err)
	}

	if false { // change to true for quick debug tracing
		txscript.SetLogWriter(os.Stdout, "trace")
		defer txscript.DisableLog()
	}

	for i, input := range tx.Inputs {
		if input.IsIssuance() {
			// TODO: implement issuance scheme
			continue
		}
		unspent := view.Output(ctx, input.Previous)
		err = engine.Prepare(unspent.Script, i)
		if err != nil {
			err = errors.Wrapf(ErrBadTx, "cannot prepare script engine to process input %d: %s", i, err)
			return errors.WithDetailf(err, "invalid script on input %d", i)
		}
		if err = engine.Execute(); err != nil {
			pkScriptStr, _ := txscript.DisasmString(unspent.Script)
			sigScriptStr, _ := txscript.DisasmString(input.SignatureScript)
			return errors.WithDetailf(ErrBadTx, "validation failed in script execution, input %d (sigscript[%s] pkscript[%s])", i, sigScriptStr, pkScriptStr)
		}
	}

	return nil
}

// txIsWellFormed checks whether tx passes context-free validation.
// If tx is well formed, it returns a nil error;
// otherwise, it returns an error describing why tx is invalid.
func txIsWellFormed(tx *bc.Tx) error {
	if len(tx.Inputs) == 0 {
		return errors.WithDetail(ErrBadTx, "inputs are missing")
	}

	// Check for duplicate inputs
	uniqueFilter := map[bc.Outpoint]bool{}
	for _, txin := range tx.Inputs {
		if txin.IsIssuance() {
			continue
		}
		if uniqueFilter[txin.Previous] {
			return errors.WithDetailf(ErrBadTx, "duplicated input for %s", txin.Previous.String())
		}
		uniqueFilter[txin.Previous] = true
	}

	// Check that every output has a valid value.
	for _, txout := range tx.Outputs {
		// Zero-value outputs are allowed for re-publishing
		// asset definition using issuance transactions.
		// Non-issuance transactions cannot have zero-value outputs.
		// If all inputs have zero value, tx therefore must have no outputs.
		// TODO: check that output asset id is an asset being issued
		if txout.Amount == 0 && !tx.HasIssuance() {
			return errors.WithDetailf(ErrBadTx, "non-issuance output value must be greater than 0")
		}
	}
	return nil
}

// validateTxBalance ensures that non-issuance transactions
// have the exact same input and output asset amounts.
func validateTxBalance(ctx context.Context, view state.ViewReader, tx *bc.Tx) error {
	parity := make(map[bc.AssetID]int64)
	issued := make(map[bc.AssetID]bool)
	for _, out := range tx.Outputs {
		parity[out.AssetID] -= int64(out.Amount)
	}
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			assetID, err := assetIDFromSigScript(in.SignatureScript)
			if err != nil {
				return err
			}
			issued[assetID] = true
			continue
		}
		unspent := view.Output(ctx, in.Previous)
		assetID := unspent.AssetID
		if assetID == (bc.AssetID{}) {
			assetID = bc.ComputeAssetID(unspent.Script, stubGenesisHash)
		}
		parity[assetID] += int64(unspent.Amount)
	}
	for asset, val := range parity {
		if val > 0 || (val < 0 && !issued[asset]) {
			return errors.WithDetailf(ErrBadTx, "amounts for asset %s are not balanced on inputs and outputs", asset)
		}
	}
	return nil
}

// ApplyTx updates the view with all the changes to the ledger
func ApplyTx(ctx context.Context, view state.View, tx *bc.Tx) error {
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			continue
		}
		o := view.Output(ctx, in.Previous)
		o.Spent = true
		view.SaveOutput(o)
	}

	for _, in := range tx.Inputs {
		// If metadata field is empty, no update of ADP takes place.
		// See https://github.com/chain-engineering/fedchain/blob/master/documentation/fedchain-specification.md#extract-asset-definition.
		if in.IsIssuance() && len(in.AssetDefinition) > 0 {
			assetID, err := assetIDFromSigScript(in.SignatureScript)
			if err != nil {
				return err
			}
			defHash := bc.HashAssetDefinition(in.AssetDefinition)
			view.SaveAssetDefinitionPointer(assetID, defHash)
		}
	}

	for i, out := range tx.Outputs {
		o := &state.Output{
			TxOutput: *out,
			Outpoint: bc.Outpoint{Hash: tx.Hash, Index: uint32(i)},
			Spent:    false,
		}
		view.SaveOutput(o)
	}

	issued := sumIssued(ctx, view, tx)
	for asset, amt := range issued {
		view.SaveIssuance(asset, amt)
	}

	return nil
}

func assetIDFromSigScript(script []byte) (bc.AssetID, error) {
	redeemScript, err := txscript.RedeemScriptFromP2SHSigScript(script)
	if err != nil {
		return bc.AssetID{}, errors.Wrap(err, "extracting redeem script from sigscript")
	}
	pkScript := txscript.RedeemToPkScript(redeemScript)
	return bc.ComputeAssetID(pkScript, [32]byte{}), nil // TODO(tessr): get genesis hash
}

// the amount of issued assets can be determined by
// the sum of outputs minus the sum of non-issuance inputs
func sumIssued(ctx context.Context, view state.ViewReader, tx *bc.Tx) map[bc.AssetID]uint64 {
	issued := make(map[bc.AssetID]uint64)
	if !tx.HasIssuance() {
		return nil
	}
	for _, out := range tx.Outputs {
		issued[out.AssetID] += out.Amount
	}
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			continue
		}
		prevout := view.Output(ctx, in.Previous)
		issued[prevout.AssetID] -= prevout.Amount
	}
	return issued
}
