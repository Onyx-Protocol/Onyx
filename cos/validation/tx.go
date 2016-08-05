package validation

import (
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"strings"

	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/cos/state"
	"chain/cos/txscript"
	"chain/errors"
)

// PriorIssuances maps a tx hash (of a tx containing an issuance) to
// the time (in Unix millis) at which it should expire from the
// issuance memory.
type PriorIssuances map[bc.Hash]uint64

var stubGenesisHash = bc.Hash{}

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// ConfirmTx validates the given transaction against the given state tree
// before it's added to a block. If tx is invalid, it returns a non-nil
// error describing why.
//
// Tx should have already been validated (with `ValidateTx`) when the tx
// was added to the pool.
//
// TODO(bobg): Combine the "tree" and "priorIssuances" arguments taken
// by ConfirmTx and ValidateTx into a single "state" object.
func ConfirmTx(tree *patricia.Tree, priorIssuances PriorIssuances, tx *bc.Tx, timestampMS uint64) error {
	if timestampMS < tx.MinTime {
		return errors.WithDetail(ErrBadTx, "block time is before transaction min time")
	}
	if tx.MaxTime > 0 && timestampMS > tx.MaxTime {
		return errors.WithDetail(ErrBadTx, "block time is after transaction max time")
	}
	for inIndex, txin := range tx.Inputs {
		if ic, ok := txin.InputCommitment.(*bc.IssuanceInputCommitment); ok {
			// txin is an issuance

			if inIndex == 0 {
				if ic.MinTimeMS > timestampMS || ic.MaxTimeMS < timestampMS {
					return errors.WithDetailf(ErrBadTx, "block time is outside issuance time window (input %d)", inIndex)
				}
				if priorIssuances != nil {
					// prune
					for txhash, expireMS := range priorIssuances {
						if timestampMS > expireMS {
							delete(priorIssuances, txhash)
						}
					}
					if _, ok2 := priorIssuances[tx.Hash]; ok2 {
						return errors.WithDetail(ErrBadTx, "duplicate issuance transaction")
					}
				}
			}
			continue
		}

		// txin is a spend

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

	for i, txin := range tx.Inputs {
		assetID := txin.AssetID()
		parity[assetID] += int64(txin.Amount())
		if txin.IsIssuance() {
			issued[assetID] = true
			if i == 0 {
				ic := txin.InputCommitment.(*bc.IssuanceInputCommitment)
				if ic.MaxTimeMS < ic.MinTimeMS {
					return errors.WithDetail(ErrBadTx, "input 0 is an issuance with maxtime < mintime")
				}
				// TODO(bobg): test that issuance maxtime - issuance mintime <= issuance window limit
			}
		}

		for j := 0; j < i; j++ {
			other := tx.Inputs[j]
			if reflect.DeepEqual(txin.InputCommitment, other.InputCommitment) {
				return errors.WithDetailf(ErrBadTx, "input %d is a duplicate of %d", j, i)
			}
		}
	}

	// Check that every output has a valid value.
	for _, txout := range tx.Outputs {
		// Transactions cannot have zero-value outputs.
		// If all inputs have zero value, tx therefore must have no outputs.
		if txout.Amount == 0 {
			return errors.WithDetailf(ErrBadTx, "output value must be greater than 0")
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
		var program []byte
		if input.IsIssuance() {
			program = input.IssuanceProgram()
		} else {
			program = input.ControlProgram()
		}
		err = engine.Prepare(program, input.InputWitness, i)
		if err != nil {
			err = errors.Wrapf(ErrBadTx, "cannot prepare VM to process input %d: %s", i, err)
			return errors.WithDetailf(err, "invalid program on input %d", i)
		}
		if err = engine.Execute(); err != nil {
			scriptStr, _ := txscript.DisasmString(program)
			hexArgs := make([]string, 0, len(input.InputWitness))
			for _, arg := range input.InputWitness {
				hexArgs = append(hexArgs, hex.EncodeToString(arg))
			}
			return errors.WithDetailf(ErrBadTx, "validation failed in script execution, input %d (args [%s] program [%s])", i, strings.Join(hexArgs, " "), scriptStr)
		}
	}
	return nil
}

// ApplyTx updates the state tree with all the changes to the ledger.
func ApplyTx(tree *patricia.Tree, priorIssuances PriorIssuances, tx *bc.Tx) error {
	for i, in := range tx.Inputs {
		if ic, ok := in.InputCommitment.(*bc.IssuanceInputCommitment); ok {
			// issuance input
			if i == 0 && priorIssuances != nil {
				priorIssuances[tx.Hash] = ic.MaxTimeMS
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
