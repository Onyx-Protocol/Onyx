package validation

import (
	"bytes"
	"context"
	"math"

	"golang.org/x/sync/errgroup"

	"chain-stealth/crypto/ca"
	"chain-stealth/errors"
	"chain-stealth/math/checked"
	"chain-stealth/protocol/bc"
	"chain-stealth/protocol/state"
	"chain-stealth/protocol/vm"
	"chain-stealth/protocol/vmutil"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

var (
	// "suberrors" for ErrBadTx
	errTxVersion              = errors.New("unknown transaction version")
	errNotYet                 = errors.New("block time is before transaction min time")
	errTooLate                = errors.New("block time is after transaction max time")
	errWrongBlockchain        = errors.New("issuance is for different blockchain")
	errTimelessIssuance       = errors.New("zero mintime or maxtime not allowed in issuance with non-empty nonce")
	errIssuanceTime           = errors.New("timestamp outside issuance input's time window")
	errDuplicateIssuance      = errors.New("duplicate issuance transaction")
	errInvalidOutput          = errors.New("invalid output")
	errNoInputs               = errors.New("inputs are missing")
	errTooManyInputs          = errors.New("number of inputs overflows uint32")
	errAllEmptyNonceIssuances = errors.New("all inputs are issuances with empty nonce fields")
	errMisorderedTime         = errors.New("positive maxtime must be >= mintime")
	errAssetVersion           = errors.New("unknown asset version")
	errInputTooBig            = errors.New("input value exceeds maximum value of int64")
	errInputSumTooBig         = errors.New("sum of inputs overflows the allowed asset amount")
	errVMVersion              = errors.New("unknown vm version")
	errDuplicateInput         = errors.New("duplicate input")
	errTooManyOutputs         = errors.New("number of outputs overflows int32")
	errEmptyOutput            = errors.New("output value must be greater than 0")
	errOutputTooBig           = errors.New("output value exceeds maximum value of int64")
	errOutputSumTooBig        = errors.New("sum of outputs overflows the allowed asset amount")
	errUnbalancedV1           = errors.New("amounts for asset are not balanced on v1 inputs and outputs")
)

func badTxErr(suberr error) error {
	err := errors.WithData(ErrBadTx, "badtx", suberr)
	err = errors.WithDetail(err, suberr.Error())
	return err
}

func badTxErrf(suberr error, f string, args ...interface{}) error {
	err := errors.WithData(ErrBadTx, "badtx", suberr)
	err = errors.WithDetailf(err, f, args...)
	return err
}

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
	if tx.Version < 1 || tx.Version > block.Version {
		return badTxErrf(errTxVersion, "unknown transaction version %d for block version %d", tx.Version, block.Version)
	}

	if block.TimestampMS < tx.MinTime {
		return badTxErr(errNotYet)
	}
	if tx.MaxTime > 0 && block.TimestampMS > tx.MaxTime {
		return badTxErr(errTooLate)
	}

	for i, txin := range tx.Inputs {
		if txin.IsIssuance() {
			if txin.AssetVersion != 1 && txin.AssetVersion != 2 {
				continue
			}
			switch inp := txin.TypedInput.(type) {
			case *bc.IssuanceInput1:
				if inp.InitialBlock != initialBlockHash {
					return badTxErr(errWrongBlockchain)
				}
			case *bc.IssuanceInput2:
				for _, c := range inp.AssetChoices {
					if c.InitialBlock != initialBlockHash {
						return badTxErr(errWrongBlockchain)
					}
				}
			}
			nonce, _ := txin.Nonce()
			if len(nonce) == 0 {
				continue
			}
			if tx.MinTime == 0 || tx.MaxTime == 0 {
				return badTxErr(errTimelessIssuance)
			}
			if block.TimestampMS < tx.MinTime || block.TimestampMS > tx.MaxTime {
				return badTxErr(errIssuanceTime)
			}
			iHash, err := tx.IssuanceHash(i)
			if err != nil {
				return err
			}
			if _, ok2 := snapshot.Issuances[iHash]; ok2 {
				return badTxErr(errDuplicateIssuance)
			}
			continue
		}

		// txin is a spend

		// Lookup the prevout in the blockchain state tree.
		k, val := state.OutputTreeItem(state.Prevout(txin))
		if !snapshot.Contains(k, val, txin.AssetVersion) {
			outpoint, ok := txin.Outpoint()
			if !ok {
				return badTxErrf(errInvalidOutput, "output for input %d is invalid", i)
			}
			return badTxErrf(errInvalidOutput, "output %s for input %d is invalid", outpoint.String(), i)
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
		return badTxErr(errNoInputs)
	}

	if len(tx.Inputs) > math.MaxInt32 {
		return badTxErr(errTooManyInputs)
	}

	// Are all inputs issuances, all with asset version 1 or 2, and all with empty nonces?
	allIssuancesWithEmptyNonces := true
	for _, txin := range tx.Inputs {
		if txin.AssetVersion != 1 && txin.AssetVersion != 2 {
			allIssuancesWithEmptyNonces = false
			break
		}
		nonce, ok := txin.Nonce()
		if !ok {
			allIssuancesWithEmptyNonces = false
			break
		}
		if len(nonce) > 0 {
			allIssuancesWithEmptyNonces = false
			break
		}
	}
	if allIssuancesWithEmptyNonces {
		return badTxErr(errAllEmptyNonceIssuances)
	}

	// Check that the transaction maximum time is greater than or equal to the
	// minimum time, if it is greater than 0.
	if tx.MaxTime > 0 && tx.MaxTime < tx.MinTime {
		return badTxErr(errMisorderedTime)
	}

	// Check that each input commitment appears only once. Also check that sums
	// of inputs and outputs balance (for asset v1), and check that both input and output sums
	// are less than 2^63 so that they don't overflow their int64 representation.
	parity := make(map[bc.AssetID]int64)
	commitments := make(map[string]int)

	// For asset v2, accumulate items for use in a call to VerifyConfidentialAssets
	var (
		v2issuances []ca.Issuance
		v2spends    []ca.Spend // contains the prevout commitments of spend inputs
		v2outputs   []ca.Output
	)

	for i, txin := range tx.Inputs {
		if (tx.Version == 1 || tx.Version == 2) && (txin.AssetVersion < 1 || txin.AssetVersion > tx.Version) {
			return badTxErrf(errAssetVersion, "unknown asset version %d in input %d for transaction version %d", txin.AssetVersion, i, tx.Version)
		}

		if txin.AssetVersion == 1 {
			assetID, _ := txin.AssetID()
			amount, _ := txin.Amount()
			if amount > math.MaxInt64 {
				return badTxErr(errInputTooBig)
			}
			sum, ok := checked.AddInt64(parity[assetID], int64(amount))
			if !ok {
				return badTxErrf(errInputSumTooBig, "adding input %d overflows the allowed asset amount", i)
			}
			parity[assetID] = sum
		}

		switch x := txin.TypedInput.(type) {
		case *bc.IssuanceInput1:
			if (tx.Version == 1 || tx.Version == 2) && x.VMVersion != 1 {
				return badTxErrf(errVMVersion, "unknown vm version %d in input %d for transaction version %d", x.VMVersion, i, tx.Version)
			}
			if txin.AssetVersion != 1 {
				continue
			}
			if len(x.Nonce) == 0 {
				continue
			}
			if tx.MinTime == 0 || tx.MaxTime == 0 {
				return badTxErr(errTimelessIssuance)
			}
		case *bc.IssuanceInput2:
			v2issuances = append(v2issuances, x)
		case *bc.SpendInput:
			if (tx.Version == 1 || tx.Version == 2) && x.VMVer() != 1 {
				return badTxErrf(errVMVersion, "unknown vm version %d in input %d for transaction version %d", x.VMVer(), i, tx.Version)
			}
			if y, ok := x.TypedOutput.(*bc.Outputv2); ok {
				v2spends = append(v2spends, y)
			}
		}
		buf := new(bytes.Buffer)
		err := txin.WriteInputCommitment(buf)
		if err != nil {
			return err
		}
		if inp, ok := commitments[string(buf.Bytes())]; ok {
			return badTxErrf(errDuplicateInput, "input %d is a duplicate of %d", i, inp)
		}
		commitments[string(buf.Bytes())] = i
	}

	if len(tx.Outputs) > math.MaxInt32 {
		return badTxErr(errTooManyOutputs)
	}

	// Check that every output has a valid value.
	for i, txout := range tx.Outputs {
		if tx.Version == 1 || tx.Version == 2 {
			if txout.AssetVersion < 1 || txout.AssetVersion > tx.Version {
				return badTxErrf(errAssetVersion, "unknown asset version %d in output %d for transaction version %d", txout.AssetVersion, i, tx.Version)
			}
			if txout.VMVer() != 1 {
				return badTxErrf(errVMVersion, "unknown vm version %d in output %d for transaction version 1", txout.VMVer(), i)
			}
		}

		switch x := txout.TypedOutput.(type) {
		case *bc.Outputv1:
			// Transactions cannot have zero-value outputs.
			// If all inputs have zero value, tx therefore must have no outputs.
			if x.AssetAmount.Amount == 0 {
				return badTxErr(errEmptyOutput)
			}
			if x.AssetAmount.Amount > math.MaxInt64 {
				return badTxErr(errOutputTooBig)
			}
			sum, ok := checked.SubInt64(parity[x.AssetAmount.AssetID], int64(x.AssetAmount.Amount))
			if !ok {
				return badTxErrf(errOutputSumTooBig, "adding output %d overflows the allowed asset amount", i)
			}
			parity[x.AssetAmount.AssetID] = sum
		case *bc.Outputv2:
			v2outputs = append(v2outputs, x)
		}
	}

	for assetID, val := range parity {
		if val != 0 {
			return badTxErrf(errUnbalancedV1, "amounts for asset %s are not balanced on v1 inputs and outputs", assetID)
		}
	}

	g, newctx := errgroup.WithContext(context.Background())
	for i := range tx.Inputs {
		i := i
		g.Go(func() error {
			err := vm.VerifyTxInput(newctx, tx, i)
			if err != nil {
				return badTxErrf(err, "validation failed in script execution, input %d", i)
			}
			return nil
		})
	}
	err := g.Wait()
	if err != nil {
		return err
	}

	if len(v2issuances) > 0 || len(v2spends) > 0 || len(v2outputs) > 0 {
		err := ca.VerifyConfidentialAssets(v2issuances, v2spends, v2outputs, tx.ExcessCommitments)
		if err != nil {
			return badTxErr(err)
		}
	}

	return nil
}

// ApplyTx updates the state tree with all the changes to the ledger.
func ApplyTx(snapshot *state.Snapshot, tx *bc.Tx) error {
	for i, in := range tx.Inputs {
		if in.IsIssuance() {
			nonce, _ := in.Nonce()
			if len(nonce) > 0 {
				iHash, err := tx.IssuanceHash(i)
				if err != nil {
					return err
				}
				snapshot.Issuances[iHash] = tx.MaxTime
			}
			continue
		}

		// Remove the consumed output from the state tree.
		outpoint, _ := in.Outpoint()
		prevoutKey := state.OutputKey(outpoint)
		err := snapshot.Delete(prevoutKey)
		if err != nil {
			return err
		}
	}

	for i, out := range tx.Outputs {
		if vmutil.IsUnspendable(out.Program()) {
			continue
		}
		// Insert new outputs into the state tree.
		o := state.NewOutput(out.TypedOutput, bc.Outpoint{Hash: tx.Hash, Index: uint32(i)})
		err := snapshot.Insert(o)
		if err != nil {
			return err
		}
	}
	return nil
}
