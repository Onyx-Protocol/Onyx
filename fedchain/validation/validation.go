package validation

import (
	"fmt"

	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

// TxIsWellFormed checks whether tx passes context-free validation.
// If tx is well formed, it returns a nil error;
// otherwise, it returns an error describing why tx is invalid.
func TxIsWellFormed(tx *bc.Tx) error {
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

// TxIsValid validates a transaction.
// If tx is valid, it returns a nil error;
// otherwise, it returns an error describing why tx is invalid.
func TxIsValid(tx *bc.Tx, view state.View, params *Params, timestamp uint64) error {
	err := TxIsWellFormed(tx)
	if err != nil {
		return err
	}

	// 1. Check time
	if tx.LockTime > timestamp {
		return fmt.Errorf("transaction lock time is in the future")
	}

	// 2a. If this is an issuance tx, simply assign zero IssuanceID and return.
	// There are no outputs to undo, so we return an empty undo object.
	// NOTE: review this when we implement import inputs.
	// Maybe we'll need to have undo ADP.
	if tx.IsIssuance() {
		for i := range tx.Outputs {
			txout := &tx.Outputs[i]
			if txout.AssetID != (bc.AssetID{}) {
				return fmt.Errorf("issuance transaction output must contain zero AssetID")
			}
			// Set Issuance ID to canonical zero value.
			// Asset is not issued until this output is spent,
			// so we cannot allow non-zero values for
			// Asset ID and Issuance ID.
			txout.IssuanceID = bc.IssuanceID{}
		}
		return nil
	}

	// 2b. Verify inputs for double-spends, color with asset ids
	// and issuance ids, extract asset definition pointers.
	for inIndex := range tx.Inputs {
		txin := &tx.Inputs[inIndex]
		unspent, err := view.Output(txin.Previous)
		if err != nil {
			return errors.Wrapf(err, "output %v", txin.Previous)
		}
		// It's possible to load a spent output here because BackedView
		// explicitly stores spent outputs in frontend to shadow unspent
		// outputs in backend.
		if unspent.Spent {
			return fmt.Errorf("output for input %d is already spent (%v)", inIndex, unspent.Outpoint)
		}

		assetID := unspent.AssetID
		issuanceID := unspent.IssuanceID

		// If it's an issuance output, it has zero AssetID and zero IssuanceID
		// We can now derive colors from the issuing output for the rest of
		// inputs and outputs.
		if assetID == (bc.AssetID{}) {
			assetID = bc.ComputeAssetID(unspent.Script, params.GenesisHash)
			issuanceID = bc.ComputeIssuanceID(unspent.Outpoint)
		}

		// Color the input too so we can access this data
		// in ValidateTxBalance and in scripts.
		txin.Value = unspent.Value
		txin.AssetID = assetID
		txin.IssuanceID = issuanceID
	}

	err = ValidateTxBalance(tx)
	if err != nil {
		return err
	}

	// TODO(erykwalder): check scripts

	return nil
}

// ApplyTx takes a validated transaction
// and applies its updates to the view.
// The returned undo objects can be used to restore
// the state to what it was before the transaction
// was applied.
func ApplyTx(tx *bc.Tx, view state.View, mod *ADPUpdates) (*UndoTx, error) {
	undo := new(UndoTx)

	// Add unspent issuance outputs to the view.
	// Store undo data
	for i := range tx.Inputs {
		if tx.Inputs[i].IsIssuance() {
			continue
		}

		unspent, err := view.Output(tx.Inputs[i].Previous)
		if err != nil {
			return nil, errors.Wrap(err, "fetching input utxo")
		}
		unspent.Spent = true
		err = view.SaveOutput(unspent)
		if err != nil {
			return nil, errors.Wrap(err, "saving input spend")
		}

		inputUndo := &UndoTxInput{
			Output: *unspent,
		}
		undo.UndoInputs = append(undo.UndoInputs, inputUndo)

		assetDef, err := NewAssetDefinition(tx, unspent, &tx.Inputs[i], uint32(i))
		if err != nil {
			return nil, err
		}

		if assetDef != nil {
			// Note 1: using asset id from the ADP because it may be
			// different than the current input's asset ID.
			// Note 2: ad_data itself may be invalid for many reasons,
			// but it also may be missing or being indirect so its status
			// won't be known until later.
			// Clients must be prepared to use ADP.verify_definition method
			// anyway when they want to trust the data.

			// Modify the pointer using the view.
			// If this input re-defines some asset definition pointer,
			// we should remember its current value.
			inputUndo.ADP, err = view.AssetDefinitionPointer(assetDef.Pointer.AssetID)
			if err != nil && err != state.ErrNotFound {
				return nil, errors.Wrap(err, "ADP for asset %v", assetDef.Pointer.AssetID)
			}

			err = view.SaveAssetDefinitionPointer(&assetDef.Pointer)
			if err != nil {
				return nil, errors.Wrap(err, "saving asset definition pointer")
			}
			// Save actual extracted data if it's present.
			if mod != nil {
				mod.Updates = append(mod.Updates, assetDef)
			}
		}
	}
	for i := range tx.Outputs {
		unspent := state.Output{
			TxOutput: tx.Outputs[i],
			Outpoint: bc.Outpoint{Hash: tx.Hash(), Index: uint32(i)},
			Spent:    false,
		}
		err := view.SaveOutput(&unspent)
		if err != nil {
			return nil, errors.Wrap(err, "saving unspent output")
		}
	}

	return undo, nil
}

// ValidateTxBalance ensures that the outputs in tx
// balance with its inputs.
// It also colors the outputs according to the inputs.
// Inputs (Value, IssuanceID and AssetID must be set)
// must already be colored by the validation process.
// It returns an error if the amounts do not balance
// or if assets are mixed.
func ValidateTxBalance(tx *bc.Tx) error {
	// TODO(erykwalder): implement this check

	return nil
}
