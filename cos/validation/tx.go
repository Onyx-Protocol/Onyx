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

// ConfirmTx validates the given transaction against the given state tree
// before it's added to a block. If tx is invalid, it returns a non-nil
// error describing why.
//
// Tx should have already been validated (with `ValidateTx`) when the tx
// was added to the pool.
func ConfirmTx(tree *patricia.Tree, tx *bc.Tx, timestampMS uint64) error {
	if timestampMS < tx.MinTime {
		return errors.WithDetail(ErrBadTx, "block time is before transaction min time")
	}
	if tx.MaxTime > 0 && timestampMS > tx.MaxTime {
		return errors.WithDetail(ErrBadTx, "block time is after transaction max time")
	}

	for inIndex, txin := range tx.Inputs {
		if txin.IsIssuance() {
			// TODO(bobg): dedupe this input against others whose [MinTime,MaxTime] windows haven't closed
			// TODO(bobg): run issuance program with args from input witness
			continue
		}

		// Lookup the prevout in the blockchain state tree.
		k, val := state.OutputTreeItem(state.Prevout(txin))
		n := tree.Lookup(k)
		if n == nil || n.Hash() != val.Value().Hash() {
			return errors.WithDetailf(ErrBadTx, "output %s for input %d is invalid", txin.Outpoint().String(), inIndex)
		}
	}
	return nil
}

// ValidateTx checks whether tx passes context-free validation:
// - inputs and outputs balance
// - no duplicate prevouts
// - input scripts pass
//
// If tx is well formed and valid, it returns a nil error; otherwise, it
// returns an error describing why tx is invalid.
func ValidateTx(tx *bc.Tx) error {
	if len(tx.Inputs) == 0 {
		return errors.WithDetail(ErrBadTx, "inputs are missing")
	}

	if tx.MaxTime > 0 && tx.MaxTime < tx.MinTime {
		return errors.WithDetail(ErrBadTx, "positive maxtime must be >= mintime")
	}

	issued := make(map[bc.AssetID]bool)
	parity := make(map[bc.AssetID]int64)
	uniqueFilter := make(map[bc.Outpoint]bool)

	for _, txin := range tx.Inputs {
		assetID := txin.AssetID()
		parity[assetID] += int64(txin.Amount())
		if txin.IsIssuance() {
			issued[assetID] = true
			// TODO(bobg): count the issuance's amount in the parity map too
			// TODO(bobg): test that issuance maxtime >= issuance mintime (if this is input 0)
			// TODO(bobg): test that issuance maxtime - issuance mintime <= issuance window limit
			continue
		}

		// Check for duplicate inputs
		// TODO(bobg): with issuance inputs too
		previous := txin.Outpoint()
		if uniqueFilter[previous] {
			return errors.WithDetailf(ErrBadTx, "duplicated input for %s", previous.String())
		}
		uniqueFilter[previous] = true
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
		if val != 0 {
			return errors.WithDetailf(ErrBadTx, "amounts for asset %s are not balanced on inputs and outputs", asset)
		}
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
		err = engine.Prepare(input.ControlProgram(), input.InputWitness, i)
		if err != nil {
			err = errors.Wrapf(ErrBadTx, "cannot prepare script engine to process input %d: %s", i, err)
			return errors.WithDetailf(err, "invalid script on input %d", i)
		}
		if err = engine.Execute(); err != nil {
			pkScriptStr, _ := txscript.DisasmString(input.ControlProgram())
			return errors.WithDetailf(ErrBadTx, "validation failed in script execution, input %d (args [%v] program [%s])", i, input.InputWitness, pkScriptStr)
		}
	}
	return nil
}

// ApplyTx updates the state tree with all the changes to the ledger.
func ApplyTx(tree *patricia.Tree, tx *bc.Tx) error {
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			// If asset definition field is empty, no update of ADP takes place.
			ad := in.AssetDefinition()
			if len(ad) > 0 {
				assetID := in.AssetID()
				defHash := bc.HashAssetDefinition(ad)
				err := tree.Insert(state.ADPTreeItem(assetID, defHash))
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
