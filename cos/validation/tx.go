package validation

import (
	"fmt"
	"os"

	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/cos/state"
	"chain/cos/txscript"
	"chain/errors"
)

var stubGenesisHash = bc.Hash{}

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// ValidateTxInputs just validates that the tx inputs are present
// and unspent in the state tree or pool.
func ValidateTxInputs(tree *patricia.Tree, poolOutputs state.OutputSet, tx *bc.Tx) error {
	for inIndex, txin := range tx.Inputs {
		if txin.IsIssuance() {
			continue
		}

		// Lookup the prevout in the blockchain state tree.
		k, val := state.OutputTreeItem(state.Prevout(txin))
		n := tree.Lookup(k)
		if n != nil {
			if n.Hash() != val.Value().Hash() {
				return errors.WithDetailf(ErrBadTx, "output %s for input %d is invalid", txin.Previous.String(), inIndex)
			}
			continue
		}

		// Check if the prevout exists in the tx pool.
		if !poolOutputs.Contains(state.Prevout(txin)) {
			return errors.WithDetailf(ErrBadTx, "output %s for input %d is invalid or already spent", txin.Previous.String(), inIndex)
		}
	}
	return nil
}

// ValidateTx validates the given transaction against the given state tree.
// If tx is invalid, it returns a non-nil error describing why.
func ValidateTx(tree *patricia.Tree, poolOutputs state.OutputSet, tx *bc.Tx, timestampMS uint64) error {
	err := txIsWellFormed(tx)
	if err != nil {
		return errors.Wrap(err, "well-formed test")
	}

	if timestampMS < tx.MinTime {
		return errors.WithDetail(ErrBadTx, "block or current time is before transaction min time")
	}

	if tx.MaxTime > 0 && timestampMS > tx.MaxTime {
		return errors.WithDetail(ErrBadTx, "block or current time is after transaction max time")
	}

	// If this is an issuance tx, check its prevout hash against the
	// previous block hash if we have one.
	// NOTE: review this when we implement import inputs.
	// Maybe we'll need to have undo ADP.
	// TODO(erykwalder): some type of uniqueness check

	err = ValidateTxInputs(tree, poolOutputs, tx)
	if err != nil {
		return errors.Wrap(err, "validating inputs")
	}

	engine, err := txscript.NewReusableEngine(&tx.TxData, txscript.StandardVerifyFlags)
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
		err = engine.Prepare(input.PrevScript, i)
		if err != nil {
			err = errors.Wrapf(ErrBadTx, "cannot prepare script engine to process input %d: %s", i, err)
			return errors.WithDetailf(err, "invalid script on input %d", i)
		}
		if err = engine.Execute(); err != nil {
			pkScriptStr, _ := txscript.DisasmString(input.PrevScript)
			sigScriptStr, _ := txscript.DisasmString(input.SignatureScript)
			return errors.WithDetailf(ErrBadTx, "validation failed in script execution, input %d (sigscript[%s] pkscript[%s])", i, sigScriptStr, pkScriptStr)
		}
	}
	return nil
}

// txIsWellFormed checks whether tx passes context-free validation:
// - inputs and outputs balance
// - no duplicate prevouts
// If tx is well formed, it returns a nil error; otherwise, it
// returns an error describing why tx is invalid.
func txIsWellFormed(tx *bc.Tx) error {
	if len(tx.Inputs) == 0 {
		return errors.WithDetail(ErrBadTx, "inputs are missing")
	}

	if tx.MaxTime > 0 && tx.MaxTime < tx.MinTime {
		return errors.WithDetail(ErrBadTx, "positive maxtime must be >= mintime")
	}

	parity := make(map[bc.AssetID]int64)
	issued := make(map[bc.AssetID]bool)
	uniqueFilter := make(map[bc.Outpoint]bool)

	for _, txin := range tx.Inputs {
		if txin.IsIssuance() {
			assetID, err := AssetIDFromSigScript(txin.SignatureScript)
			if err != nil {
				return err
			}
			issued[assetID] = true
			continue
		}
		assetID := txin.AssetAmount.AssetID
		if assetID == (bc.AssetID{}) {
			assetID = bc.ComputeAssetID(txin.PrevScript, stubGenesisHash)
		}
		parity[assetID] += int64(txin.AssetAmount.Amount)

		// Check for duplicate inputs
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
		if txout.Amount == 0 && !issued[txout.AssetID] {
			return errors.WithDetailf(ErrBadTx, "non-issuance output value must be greater than 0")
		}
		parity[txout.AssetID] -= int64(txout.Amount)
	}

	for asset, val := range parity {
		if val > 0 || (val < 0 && !issued[asset]) {
			return errors.WithDetailf(ErrBadTx, "amounts for asset %s are not balanced on inputs and outputs", asset)
		}
	}
	return nil
}

// ApplyTx updates the state tree with all the changes to the ledger.
func ApplyTx(tree *patricia.Tree, tx *bc.Tx) error {
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			// If asset definition field is empty, no update of ADP takes place.
			if len(in.AssetDefinition) > 0 {
				assetID, err := AssetIDFromSigScript(in.SignatureScript)
				if err != nil {
					return err
				}
				defHash := bc.HashAssetDefinition(in.AssetDefinition)
				err = tree.Insert(state.ADPTreeItem(assetID, defHash))
				if err != nil {
					return err
				}
			}
			continue
		}

		// Remove the consumed output from the state tree.
		prevoutKey, _ := state.OutputTreeItem(state.Prevout(in))
		err := tree.Delete(prevoutKey)
		if err != nil {
			return err
		}
	}

	for i, out := range tx.Outputs {
		if txscript.IsUnspendable(out.ControlProgram) {
			continue
		}
		// Insert new outputs into the state tree.
		o := state.NewOutput(*out, bc.Outpoint{Hash: tx.Hash, Index: uint32(i)})
		err := tree.Insert(state.OutputTreeItem(o))
		if err != nil {
			return err
		}
	}
	return nil
}

// AssetIDFromSigScript takes an issuance sigscript and computes the
// associated Asset ID from it.
func AssetIDFromSigScript(script []byte) (bc.AssetID, error) {
	redeemScript, err := txscript.RedeemScriptFromP2SHSigScript(script)
	if err != nil {
		return bc.AssetID{}, errors.Wrap(err, "extracting redeem script from sigscript")
	}
	pkScript := txscript.RedeemToPkScript(redeemScript)
	return bc.ComputeAssetID(pkScript, [32]byte{}), nil // TODO(tessr): get genesis hash
}
