package validation

import (
	"bytes"
	"math"

	"chain/errors"
	"chain/math/checked"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
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

func badTxErr(err error) error {
	err = errors.WithData(err, "badtx", err)
	err = errors.WithDetail(err, err.Error())
	return errors.Sub(ErrBadTx, err)
}

func badTxErrf(err error, f string, args ...interface{}) error {
	err = errors.WithData(err, "badtx", err)
	err = errors.WithDetailf(err, f, args...)
	return errors.Sub(ErrBadTx, err)
}

// ConfirmTx validates the given transaction against the given state tree
// before it's added to a block. If tx is invalid, it returns a non-nil
// error describing why.
//
// Tx must already have undergone the well-formedness check in
// CheckTxWellFormed. This should have happened when the tx was added
// to the pool.
//
// ConfirmTx must not mutate the snapshot.
func ConfirmTx(snapshot *state.Snapshot, initialBlockHash bc.Hash, blockVersion, blockTimestampMS uint64, tx *bc.Tx) error {
	if tx.Version < 1 || tx.Version > blockVersion {
		return badTxErrf(errTxVersion, "unknown transaction version %d for block version %d", tx.Version, blockVersion)
	}

	if blockTimestampMS < tx.MinTime {
		return badTxErr(errNotYet)
	}
	if tx.MaxTime > 0 && blockTimestampMS > tx.MaxTime {
		return badTxErr(errTooLate)
	}

	for i, txin := range tx.Inputs {
		if ii, ok := txin.TypedInput.(*bc.IssuanceInput); ok {
			if txin.AssetVersion != 1 {
				continue
			}
			if ii.InitialBlock != initialBlockHash {
				return badTxErr(errWrongBlockchain)
			}
			if len(ii.Nonce) == 0 {
				continue
			}
			if tx.MinTime == 0 || tx.MaxTime == 0 {
				return badTxErr(errTimelessIssuance)
			}
			if blockTimestampMS < tx.MinTime || blockTimestampMS > tx.MaxTime {
				return badTxErr(errIssuanceTime)
			}
			iHash := tx.IssuanceHash(uint32(i))
			if _, ok2 := snapshot.Issuances[iHash]; ok2 {
				return badTxErr(errDuplicateIssuance)
			}
			continue
		}

		// txin is a spend

		spentOutputID, err := txin.SpentOutputID()
		if err != nil {
			return badTxErrf(errInvalidOutput, "could not compute output id for input %d", i)
		}

		// Lookup the prevout in the blockchain state tree.
		if !snapshot.Tree.Contains(spentOutputID.Bytes()) {
			return badTxErrf(errInvalidOutput, "output %s for input %d is invalid", spentOutputID, i)
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
		return badTxErr(errAllEmptyNonceIssuances)
	}

	// Check that the transaction maximum time is greater than or equal to the
	// minimum time, if it is greater than 0.
	if tx.MaxTime > 0 && tx.MaxTime < tx.MinTime {
		return badTxErr(errMisorderedTime)
	}

	// Check that each input commitment appears only once. Also check that sums
	// of inputs and outputs balance, and check that both input and output sums
	// are less than 2^63 so that they don't overflow their int64 representation.
	parity := make(map[bc.AssetID]int64)
	commitments := make(map[string]int)

	for i, txin := range tx.Inputs {
		if tx.Version == 1 && txin.AssetVersion != 1 {
			return badTxErrf(errAssetVersion, "unknown asset version %d in input %d for transaction version %d", txin.AssetVersion, i, tx.Version)
		}

		assetID := txin.AssetID()

		if txin.Amount() > math.MaxInt64 {
			return badTxErr(errInputTooBig)
		}

		sum, ok := checked.AddInt64(parity[assetID], int64(txin.Amount()))
		if !ok {
			return badTxErrf(errInputSumTooBig, "adding input %d overflows the allowed asset amount", i)
		}
		parity[assetID] = sum

		switch x := txin.TypedInput.(type) {
		case *bc.IssuanceInput:
			if tx.Version == 1 && x.VMVersion != 1 {
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
		case *bc.SpendInput:
			if tx.Version == 1 && x.VMVersion != 1 {
				return badTxErrf(errVMVersion, "unknown vm version %d in input %d for transaction version %d", x.VMVersion, i, tx.Version)
			}
		}

		buf := new(bytes.Buffer)
		txin.WriteInputCommitment(buf, bc.SerTxHash)
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
		if tx.Version == 1 {
			if txout.AssetVersion != 1 {
				return badTxErrf(errAssetVersion, "unknown asset version %d in output %d for transaction version %d", txout.AssetVersion, i, tx.Version)
			}
			if txout.VMVersion != 1 {
				return badTxErrf(errVMVersion, "unknown vm version %d in output %d for transaction version %d", txout.VMVersion, i, tx.Version)
			}
		}

		// Transactions cannot have zero-value outputs.
		// If all inputs have zero value, tx therefore must have no outputs.
		if txout.Amount == 0 {
			return badTxErr(errEmptyOutput)
		}

		if txout.Amount > math.MaxInt64 {
			return badTxErr(errOutputTooBig)
		}

		sum, ok := checked.SubInt64(parity[txout.AssetID], int64(txout.Amount))
		if !ok {
			return badTxErrf(errOutputSumTooBig, "adding output %d overflows the allowed asset amount", i)
		}
		parity[txout.AssetID] = sum
	}

	for assetID, val := range parity {
		if val != 0 {
			return badTxErrf(errUnbalancedV1, "amounts for asset %s are not balanced on v1 inputs and outputs", assetID)
		}
	}

	for i, inp := range tx.Inputs {
		var (
			prog bc.Program
			args [][]byte
		)
		switch inp := inp.TypedInput.(type) {
		case *bc.IssuanceInput:
			prog = bc.Program{VMVersion: inp.VMVersion, Code: inp.IssuanceProgram}
			args = inp.Arguments
		case *bc.SpendInput:
			prog = bc.Program{VMVersion: inp.VMVersion, Code: inp.ControlProgram}
			args = inp.Arguments
		}
		err := vm.Verify(bc.NewTxVMContext(tx, uint32(i), prog, args))
		if err != nil {
			return badTxErrf(err, "validation failed in script execution, input %d", i)
		}
	}

	return nil
}

// ApplyTx updates the state tree with all the changes to the ledger.
func ApplyTx(snapshot *state.Snapshot, tx *bc.Tx) error {
	for i, in := range tx.Inputs {
		if ii, ok := in.TypedInput.(*bc.IssuanceInput); ok {
			if len(ii.Nonce) > 0 {
				iHash := tx.IssuanceHash(uint32(i))
				snapshot.Issuances[iHash] = tx.MaxTime
			}
			continue
		}

		// Remove the consumed output from the state tree.
		uid, err := in.SpentOutputID()
		if err != nil {
			return err
		}
		snapshot.Tree.Delete(uid.Bytes())
	}

	for i, out := range tx.Outputs {
		if vmutil.IsUnspendable(out.ControlProgram) {
			continue
		}
		// Insert new outputs into the state tree.
		outputID := tx.OutputID(uint32(i))
		err := snapshot.Tree.Insert(outputID[:])
		if err != nil {
			return err
		}
	}
	return nil
}
