package validation

import (
	"bytes"
	"encoding/hex"
	"math"
	"strings"

	"chain/errors"
	"chain/math/checked"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
)

var (
	// ErrBadTx is returned for transactions failing validation
	ErrBadTx = errors.New("invalid transaction")

	// ErrFalseVMResult is one of the ways for a transaction to fail validation
	ErrFalseVMResult = errors.New("false VM result")
)

// ConfirmTx validates the given transaction against the given state tree
// before it's added to a block. If tx is invalid, it returns a non-nil
// error describing why.
//
// Tx must already have undergone the well-formedness check in
// CheckTxWellFormed. This should have happened when the tx was added
// to the pool.
//
// ConfirmTx must not mutate the snapshot or the block.
func ConfirmTx(snapshot *state.Snapshot, initialBlockHash bc.Hash, block *bc.Block, tx *bc.Tx) error {
	if block.Version == 1 && tx.Version != 1 {
		return errors.WithDetailf(ErrBadTx, "unknown transaction version %d for block version 1", tx.Version)
	}

	if block.TimestampMS < tx.MinTime {
		return errors.WithDetail(ErrBadTx, "block time is before transaction min time")
	}
	if tx.MaxTime > 0 && block.TimestampMS > tx.MaxTime {
		return errors.WithDetail(ErrBadTx, "block time is after transaction max time")
	}

	for i, txin := range tx.Inputs {
		if ii, ok := txin.TypedInput.(*bc.IssuanceInput); ok {
			if txin.AssetVersion != 1 {
				continue
			}
			if ii.InitialBlock != initialBlockHash {
				return errors.WithDetail(ErrBadTx, "issuance is for different blockchain")
			}
			if len(ii.Nonce) == 0 {
				continue
			}
			if block.TimestampMS < tx.MinTime || block.TimestampMS > tx.MaxTime {
				return errors.WithDetail(ErrBadTx, "timestamp outside issuance input's time window")
			}
			iHash, err := tx.IssuanceHash(i)
			if err != nil {
				return err
			}
			if _, ok2 := snapshot.Issuances[iHash]; ok2 {
				return errors.WithDetail(ErrBadTx, "duplicate issuance transaction")
			}
			continue
		}

		// txin is a spend

		// Lookup the prevout in the blockchain state tree.
		k, val := state.OutputTreeItem(state.Prevout(txin))
		if !snapshot.Tree.Contains(k, val) {
			return errors.WithDetailf(ErrBadTx, "output %s for input %d is invalid", txin.Outpoint().String(), i)
		}
	}
	return nil
}

// CheckTxWellFormed checks whether tx is "well-formed" (the
// context-free phase of validation):
// - inputs and outputs balance
// - no duplicate input commitments
// - input scripts pass
//
// Result is nil for well-formed transactions, ErrBadTx with
// supporting detail otherwise.
func CheckTxWellFormed(tx *bc.Tx) error {
	if len(tx.Inputs) == 0 {
		return errors.WithDetail(ErrBadTx, "inputs are missing")
	}

	if len(tx.Inputs) > math.MaxInt32 {
		return errors.WithDetail(ErrBadTx, "number of inputs overflows uint32")
	}

	// Are all inputs issuances, all with asset version 1, and all with empty nonces?
	allIssuancesWithEmptyNonces := true
	for _, txin := range tx.Inputs {
		if txin.AssetVersion != 1 {
			allIssuancesWithEmptyNonces = false
			break
		}
		ii, ok := txin.TypedInput.(*bc.IssuanceInput)
		if !ok {
			allIssuancesWithEmptyNonces = false
			break
		}
		if len(ii.Nonce) > 0 {
			allIssuancesWithEmptyNonces = false
			break
		}
	}
	if allIssuancesWithEmptyNonces {
		return errors.WithDetail(ErrBadTx, "all inputs are issuances with empty nonce fields")
	}

	// Check that the transaction maximum time is greater than or equal to the
	// minimum time, if it is greater than 0.
	if tx.MaxTime > 0 && tx.MaxTime < tx.MinTime {
		return errors.WithDetail(ErrBadTx, "positive maxtime must be >= mintime")
	}

	// Check that each input commitment appears only once. Also check that sums
	// of inputs and outputs balance, and check that both input and output sums
	// are less than 2^63 so that they don't overflow their int64 representation.
	parity := make(map[bc.AssetID]int64)
	commitments := make(map[string]int)

	for i, txin := range tx.Inputs {
		if tx.Version == 1 && txin.AssetVersion != 1 {
			return errors.WithDetailf(ErrBadTx, "unknown asset version %d in input %d for transaction version 1", txin.AssetVersion, i)
		}

		assetID := txin.AssetID()

		if txin.Amount() > math.MaxInt64 {
			return errors.WithDetail(ErrBadTx, "input value exceeds maximum value of int64")
		}

		sum, ok := checked.AddInt64(parity[assetID], int64(txin.Amount()))
		if !ok {
			return errors.WithDetailf(ErrBadTx, "adding input %d overflows the allowed asset amount", i)
		}
		parity[assetID] = sum

		switch x := txin.TypedInput.(type) {
		case *bc.IssuanceInput:
			if tx.Version == 1 && x.VMVersion != 1 {
				return errors.WithDetailf(ErrBadTx, "unknown vm version %d in input %d for transaction version 1", x.VMVersion, i)
			}
			if txin.AssetVersion != 1 {
				continue
			}
			if len(x.Nonce) == 0 {
				continue
			}
			if tx.MinTime == 0 || tx.MaxTime == 0 {
				return errors.WithDetail(ErrBadTx, "issuance input with unbounded time window")
			}
		case *bc.SpendInput:
			if tx.Version == 1 && x.VMVersion != 1 {
				return errors.WithDetailf(ErrBadTx, "unknown vm version %d in input %d for transaction version 1", x.VMVersion, i)
			}
		}

		buf := new(bytes.Buffer)
		txin.WriteInputCommitment(buf)
		if inp, ok := commitments[string(buf.Bytes())]; ok {
			return errors.WithDetailf(ErrBadTx, "input %d is a duplicate of %d", i, inp)
		}
		commitments[string(buf.Bytes())] = i
	}

	if len(tx.Outputs) > math.MaxInt32 {
		return errors.WithDetail(ErrBadTx, "number of outputs overflows int32")
	}

	// Check that every output has a valid value.
	for i, txout := range tx.Outputs {
		if tx.Version == 1 {
			if txout.AssetVersion != 1 {
				return errors.WithDetailf(ErrBadTx, "unknown asset version %d in output %d for transaction version 1", txout.AssetVersion, i)
			}
			if txout.VMVersion != 1 {
				return errors.WithDetailf(ErrBadTx, "unknown vm version %d in output %d for transaction version 1", txout.VMVersion, i)
			}
		}

		// Transactions cannot have zero-value outputs.
		// If all inputs have zero value, tx therefore must have no outputs.
		if txout.Amount == 0 {
			return errors.WithDetail(ErrBadTx, "output value must be greater than 0")
		}

		if txout.Amount > math.MaxInt64 {
			return errors.WithDetail(ErrBadTx, "output value exceeds maximum value of int64")
		}

		sum, ok := checked.SubInt64(parity[txout.AssetID], int64(txout.Amount))
		if !ok {
			return errors.WithDetailf(ErrBadTx, "adding output %d overflows the allowed asset amount", i)
		}
		parity[txout.AssetID] = sum
	}

	for asset, val := range parity {
		if val != 0 {
			return errors.WithDetailf(ErrBadTx, "amounts for asset %s are not balanced on inputs and outputs", asset)
		}
	}

	if len(tx.Inputs) > math.MaxInt32 {
		return errors.WithDetail(ErrBadTx, "number of inputs overflows int32")
	}

	for i := range tx.Inputs {
		ok, err := vm.VerifyTxInput(tx, i)
		if err == nil && !ok {
			err = ErrFalseVMResult
		}
		if err != nil {
			input := tx.Inputs[i]
			var program []byte
			if input.IsIssuance() {
				program = input.IssuanceProgram()
			} else {
				program = input.ControlProgram()
			}
			scriptStr, e := vm.Disassemble(program)
			if e != nil {
				scriptStr = "disassembly failed: " + e.Error()
			}
			args := input.Arguments()
			hexArgs := make([]string, 0, len(args))
			for _, arg := range args {
				hexArgs = append(hexArgs, hex.EncodeToString(arg))
			}
			return errors.WithDetailf(ErrBadTx, "validation failed in script execution, input %d (program [%s] args [%s]): %s", i, scriptStr, strings.Join(hexArgs, " "), err)
		}
	}
	return nil
}

// ApplyTx updates the state tree with all the changes to the ledger.
func ApplyTx(snapshot *state.Snapshot, tx *bc.Tx) error {
	for i, in := range tx.Inputs {
		if ii, ok := in.TypedInput.(*bc.IssuanceInput); ok {
			if len(ii.Nonce) > 0 {
				iHash, err := tx.IssuanceHash(i)
				if err != nil {
					return err
				}
				snapshot.Issuances[iHash] = tx.MaxTime
			}
			continue
		}

		// Remove the consumed output from the state tree.
		prevoutKey := state.OutputKey(in.Outpoint())
		err := snapshot.Tree.Delete(prevoutKey)
		if err != nil {
			return err
		}
	}

	for i, out := range tx.Outputs {
		if vmutil.IsUnspendable(out.ControlProgram) {
			continue
		}
		// Insert new outputs into the state tree.
		o := state.NewOutput(*out, bc.Outpoint{Hash: tx.Hash, Index: uint32(i)})
		err := snapshot.Tree.Insert(state.OutputTreeItem(o))
		if err != nil {
			return err
		}
	}
	return nil
}
