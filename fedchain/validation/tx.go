package validation

import (
	"fmt"

	"golang.org/x/net/context"

	"chain/crypto/hash256"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
)

var stubGenesisHash = bc.Hash{}

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
			return fmt.Errorf("output for input %d is invalid or already spent (%v)", inIndex, txin.Previous)
		}
	}
	return nil
}

// ValidateTx validates the given transaction
// against the given state and applies its
// changes to the view.
// If tx is invalid,
// it returns a non-nil error describing why.
func ValidateTx(ctx context.Context, view state.View, tx *bc.Tx, timestamp uint64, prevHash *bc.Hash) error {
	err := txIsWellFormed(tx)
	if err != nil {
		return err
	}

	// Check time
	if tx.LockTime > timestamp {
		return fmt.Errorf("transaction lock time is in the future")
	}

	// If this is an issuance tx, check its prevout hash against the
	// previous block hash if we have one.
	// There are no outputs to undo, so we return an empty undo object.
	// NOTE: review this when we implement import inputs.
	// Maybe we'll need to have undo ADP.
	if tx.IsIssuance() {
		if prevHash != nil && *prevHash != tx.Inputs[0].Previous.Hash {
			return fmt.Errorf("issuance prev block hash mismatch")
		}

		// TODO(erykwalder): check outputs once utxos aren't tied to manager nodes
		return ApplyTx(ctx, view, tx)
	}

	err = ValidateTxInputs(ctx, view, tx)
	if err != nil {
		return err
	}

	err = validateTxBalance(ctx, view, tx)
	if err != nil {
		return err
	}

	engine, err := txscript.NewReusableEngine(ctx, view, tx, txscript.StandardVerifyFlags)
	if err != nil {
		return fmt.Errorf("cannot create script engine: %s", err)
	}
	for i, input := range tx.Inputs {
		unspent := view.Output(ctx, input.Previous)
		err = engine.Prepare(unspent.Script, i)
		if err != nil {
			return fmt.Errorf("cannot prepare script engine to process input %d: %s", i, err)
		}
		if err = engine.Execute(); err != nil {
			return fmt.Errorf("cannot validate transaction: %s", err)
		}
	}

	return ApplyTx(ctx, view, tx)
}

// txIsWellFormed checks whether tx passes context-free validation.
// If tx is well formed, it returns a nil error;
// otherwise, it returns an error describing why tx is invalid.
func txIsWellFormed(tx *bc.Tx) error {
	if len(tx.Inputs) == 0 {
		return errors.New("inputs are missing")
	}

	// Special rules for the issuance transaction.
	// Issuance transaction must also reference previous block hash,
	// but we can verify that only within CheckBlock method.
	if tx.IsIssuance() && len(tx.Inputs) != 1 {
		return errors.New("issuance tx has more than one input")
	}

	// Check for duplicate inputs
	uniqueFilter := map[bc.Outpoint]bool{}
	for _, txin := range tx.Inputs {
		if uniqueFilter[txin.Previous] {
			return fmt.Errorf("input is duplicate: %s", txin.Previous.String())
		}
		uniqueFilter[txin.Previous] = true
	}

	// Check that every output has a valid value.
	for _, txout := range tx.Outputs {
		// Zero-value outputs are allowed for re-publishing
		// asset definition using issuance transactions.
		// Non-issuance transactions cannot have zero-value outputs.
		// If all inputs have zero value, tx therefore must have no outputs.
		if txout.Value == 0 && !tx.IsIssuance() {
			return fmt.Errorf("non-issuance output value must be > 0")
		}
	}
	return nil
}

// validateTxBalance ensures that non-issuance transactions
// have the exact same input and output asset amounts.
func validateTxBalance(ctx context.Context, view state.View, tx *bc.Tx) error {
	parity := make(map[bc.AssetID]uint64)
	for _, out := range tx.Outputs {
		parity[out.AssetID] -= out.Value
	}
	for _, in := range tx.Inputs {
		unspent := view.Output(ctx, in.Previous)
		assetID := unspent.AssetID
		if assetID == (bc.AssetID{}) {
			assetID = bc.ComputeAssetID(unspent.Script, stubGenesisHash)
		}
		parity[assetID] += unspent.Value
	}
	for _, val := range parity {
		if val != 0 {
			return errors.New("transaction does not have input output parity")
		}
	}
	return nil
}

// ApplyTx updates the view with all the changes to the ledger
func ApplyTx(ctx context.Context, view state.View, tx *bc.Tx) error {
	if !tx.IsIssuance() {
		for _, in := range tx.Inputs {
			o := view.Output(ctx, in.Previous)
			o.Spent = true
			view.SaveOutput(o)
		}
	}

	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			redeemScript, err := txscript.RedeemScriptFromP2SHSigScript(in.SignatureScript)
			if err != nil {
				return errors.Wrap(err, "extracting redeem script from sigscript")
			}
			pkScript := txscript.RedeemToPkScript(redeemScript)
			assetID := bc.ComputeAssetID(pkScript, [32]byte{}) // TODO(tessr): get genesis hash
			definition := in.AssetDefinition
			defHash := hash256.Sum(definition)
			adp := &bc.AssetDefinitionPointer{
				AssetID:        assetID,
				DefinitionHash: defHash,
			}
			view.SaveAssetDefinitionPointer(adp)
		}
	}

	for i, out := range tx.Outputs {
		o := &state.Output{
			TxOutput: *out,
			Outpoint: bc.Outpoint{Hash: tx.Hash(), Index: uint32(i)},
			Spent:    false,
		}
		view.SaveOutput(o)
	}

	return nil
}
