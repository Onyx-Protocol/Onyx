package validation

import (
	"math"

	"chain/errors"
	"chain/math/checked"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/vm"
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
func ConfirmTx(snapshot *state.Snapshot, initialBlockHash bc.Hash, blockVersion, blockTimestampMS uint64, tx *bc.TxEntries) error {
	if tx.Version() < 1 || tx.Version() > blockVersion {
		return badTxErrf(errTxVersion, "unknown transaction version %d for block version %d", tx.Version, blockVersion)
	}

	if blockTimestampMS < tx.MinTimeMS() {
		return badTxErr(errNotYet)
	}
	if tx.MaxTimeMS() > 0 && blockTimestampMS > tx.MaxTimeMS() {
		return badTxErr(errTooLate)
	}

	for i, inp := range tx.TxInputs {
		switch inp := inp.(type) {
		case *bc.Issuance:
			if inp.InitialBlockID() != initialBlockHash {
				return badTxErr(errWrongBlockchain)
			}
			// xxx nonce/timerange check (already done in checktxwellformed)?
			if blockTimestampMS < tx.MinTimeMS() || blockTimestampMS > tx.MaxTimeMS() {
				return badTxErr(errIssuanceTime)
			}
			id := tx.TxInputIDs[i]
			if _, ok := snapshot.Issuances[id]; ok {
				return badTxErr(errDuplicateIssuance)
			}

		case *bc.Spend:
			if !snapshot.Tree.Contains(inp.SpentOutputID().Bytes()) {
				return badTxErrf(errInvalidOutput, "output %s for input %d is invalid", inp.SpentOutputID(), i)
			}
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
func CheckTxWellFormed(tx *bc.TxEntries) error {
	if len(tx.TxInputs) == 0 {
		return badTxErr(errNoInputs)
	}

	if len(tx.TxInputs) > math.MaxInt32 {
		return badTxErr(errTooManyInputs)
	}

	// Are all inputs issuances, and all with empty nonces?
	allIssuancesWithEmptyNonces := true
	for _, inp := range tx.TxInputs {
		if inp, ok := inp.(*bc.Issuance); ok {
			if (inp.AnchorID() != bc.Hash{}) { // xxx is this the txentries analog of "empty nonce"?
				allIssuancesWithEmptyNonces = false
				break
			}
		} else {
			allIssuancesWithEmptyNonces = false
			break
		}
	}
	if allIssuancesWithEmptyNonces {
		return badTxErr(errAllEmptyNonceIssuances)
	}

	// Check that the transaction maximum time is greater than or equal to the
	// minimum time, if it is greater than 0.
	if tx.MaxTimeMS() > 0 && tx.MaxTimeMS() < tx.MinTimeMS() {
		return badTxErr(errMisorderedTime)
	}

	// Check that each input appears only once. Also check that sums of
	// inputs and outputs balance, and check that both input and output
	// sums are less than 2^63 so that they don't overflow their int64
	// representation.
	parity := make(map[bc.AssetID]int64)

	for i, inpID := range tx.TxInputIDs {
		for j := i + 1; j < len(tx.TxInputIDs); j++ {
			if inpID == tx.TxInputIDs[j] {
				return badTxErrf(errDuplicateInput, "input %d is a duplicate of input %d", j, i)
			}
		}
	}

	for i, inp := range tx.TxInputs {
		var (
			assetID   bc.AssetID
			amount    uint64
			vmVersion uint64
		)

		switch inp := inp.(type) {
		case *bc.Issuance:
			assetID = inp.AssetID()
			amount = inp.Amount()
			vmVersion = inp.IssuanceProgram().VMVersion
			// xxx nonce/timerange checking

		case *bc.Spend:
			assetID = inp.AssetID()
			amount = inp.Amount()
			vmVersion = inp.ControlProgram().VMVersion

		default:
			// xxx error
		}

		if amount > math.MaxInt64 {
			return badTxErr(errInputTooBig)
		}

		if tx.Version() == 1 && vmVersion != 1 {
			return badTxErrf(errVMVersion, "unknown vm version %d in input %d for transaction version %d", vmVersion, i, tx.Version())
		}

		sum, ok := checked.AddInt64(parity[assetID], int64(amount))
		if !ok {
			return badTxErrf(errInputSumTooBig, "adding input %d overflows the allowed asset amount", i)
		}
		parity[assetID] = sum
	}

	if len(tx.Results) > math.MaxInt32 {
		return badTxErr(errTooManyOutputs)
	}

	// Check that every output has a valid value.
	for i, res := range tx.Results {
		var (
			assetID bc.AssetID
			amount  uint64
		)

		switch res := res.(type) {
		case *bc.Output:
			vmVersion := res.ControlProgram().VMVersion
			if tx.Version() == 1 && vmVersion != 1 {
				return badTxErrf(errVMVersion, "unknown vm version %d in output %d for transaction version %d", vmVersion, i, tx.Version())
			}
			assetID = res.AssetID()
			amount = res.Amount()

		case *bc.Retirement:
			assetID = res.AssetID()
			amount = res.Amount()

		default:
			// xxx error
		}

		// Transactions cannot have zero-value outputs.
		// If all inputs have zero value, tx therefore must have no outputs.
		if amount == 0 {
			return badTxErr(errEmptyOutput)
		}

		if amount > math.MaxInt64 {
			return badTxErr(errOutputTooBig)
		}

		sum, ok := checked.SubInt64(parity[assetID], int64(amount))
		if !ok {
			return badTxErrf(errOutputSumTooBig, "adding output %d overflows the allowed asset amount", i)
		}
		parity[assetID] = sum
	}

	for assetID, val := range parity {
		if val != 0 {
			return badTxErrf(errUnbalancedV1, "amounts for asset %s are not balanced on v1 inputs and outputs", assetID)
		}
	}

	for i := range tx.TxInputs {
		err := vm.VerifyTxInput(tx, uint32(i))
		if err != nil {
			return badTxErrf(err, "validation failed in script execution, input %d", i)
		}
	}

	return nil
}

// ApplyTx updates the state tree with all the changes to the ledger.
func ApplyTx(snapshot *state.Snapshot, tx *bc.TxEntries) error {
	for i, inp := range tx.TxInputs {
		switch inp := inp.(type) {
		case *bc.Issuance:
			id := tx.TxInputIDs[i]
			snapshot.Issuances[id] = tx.MaxTimeMS() // xxx or the max time from the anchor timerange?

		case *bc.Spend:
			// Remove the consumed output from the state tree.
			snapshot.Tree.Delete(inp.SpentOutputID().Bytes())
		}
	}

	for i, res := range tx.Results {
		if _, ok := res.(*bc.Output); ok {
			err := snapshot.Tree.Insert(tx.ResultID(uint32(i)).Bytes())
			if err != nil {
				return err
			}
		}
	}

	return nil
}
