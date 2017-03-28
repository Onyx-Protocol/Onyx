package bc

import "chain/errors"

type (
	// ValidChecker can check its validity with respect to a given
	// validation state.
	validChecker interface {
		// CheckValid checks the entry for validity w.r.t. the given
		// validation state.
		checkValid(*validationState) error
	}

	validationState struct {
		// The ID of the blockchain
		blockchainID Hash

		// The enclosing transaction object
		tx *TxEntries

		// The ID of the nearest enclosing entry
		entryID Hash

		// The source position, for validating ValueSources
		sourcePos uint64

		// The destination position, for validating ValueDestinations
		destPos uint64
	}
)

var (
	errBadTimeRange          = errors.New("bad time range")
	errEmptyResults          = errors.New("transaction has no results")
	errEntryType             = errors.New("invalid entry type")
	errMismatchedAssetID     = errors.New("mismatched asset id")
	errMismatchedBlock       = errors.New("mismatched block")
	errMismatchedMerkleRoot  = errors.New("mismatched merkle root")
	errMismatchedReference   = errors.New("mismatched reference")
	errMismatchedValue       = errors.New("mismatched value")
	errMisorderedBlockHeight = errors.New("misordered block height")
	errMisorderedBlockTime   = errors.New("misordered block time")
	errNoPrevBlock           = errors.New("no previous block")
	errNoSource              = errors.New("no source for value")
	errNonemptyExtHash       = errors.New("non-empty extension hash")
	errOverflow              = errors.New("arithmetic overflow/underflow")
	errPosition              = errors.New("invalid source or destination position")
	errTxVersion             = errors.New("invalid transaction version")
	errUnbalanced            = errors.New("unbalanced")
	errUntimelyTransaction   = errors.New("block timestamp outside transaction time range")
	errVersionRegression     = errors.New("version regression")
	errWrongBlockchain       = errors.New("wrong blockchain")
	errZeroTime              = errors.New("timerange has one or two bounds set to zero")
)

// ValidateTx validates a transaction.
func ValidateTx(tx *TxEntries, initialBlockID Hash) error {
	vs := &validationState{
		blockchainID: initialBlockID,
	}
	return tx.checkValid(vs)
}

// ValidateBlock validates a block and the transactions within.
// The consensus program is executed only if prev is non-nil and
// runProg is true.
func ValidateBlock(b, prev *BlockEntries, initialBlockID Hash, runProg bool) error {
	if prev == nil {
		if b.Body.Height != 1 {
			return errors.WithDetailf(errNoPrevBlock, "height %d", b.Body.Height)
		}
	} else {
		if b.Body.Version < prev.Body.Version {
			return errors.WithDetailf(errVersionRegression, "previous block verson %d, current block version %d", prev.Body.Version, b.Body.Version)
		}

		if b.Body.Height != prev.Body.Height+1 {
			return errors.WithDetailf(errMisorderedBlockHeight, "previous block height %d, current block height %d", prev.Body.Height, b.Body.Height)
		}

		if prev.ID != b.Body.PreviousBlockID {
			return errors.WithDetailf(errMismatchedBlock, "previous block ID %x, current block wants %x", prev.ID[:], b.Body.PreviousBlockID[:])
		}

		if b.Body.TimestampMS <= prev.Body.TimestampMS {
			return errors.WithDetailf(errMisorderedBlockTime, "previous block time %d, current block time %d", prev.Body.TimestampMS, b.Body.TimestampMS)
		}

		if runProg {
			vmContext := NewBlockVMContext(b, prev.Body.NextConsensusProgram, b.Witness.Arguments)
			err := vm.Verify(vmContext)
			if err != nil {
				return errors.Wrap(err, "evaluating previous block's next consensus program")
			}
		}
	}

	vs := &validationState{
		blockchainID: initialBlockID,
		entryID:      b.ID,
	}
	err := b.BlockHeaderEntry.checkValid(vs)
	if err != nil {
		return err
	}

	for i, tx := range b.Transactions {
		if b.Body.Version == 1 && tx.Body.Version != 1 {
			return errors.WithDetailf(errTxVersion, "block version %d, transaction version %d", b.Body.Version, tx.Body.Version)
		}
		if tx.Body.MaxTimeMS > 0 && b.Body.TimestampMS > tx.Body.MaxTimeMS {
			return errors.WithDetailf(errUntimelyTransaction, "block timestamp %d, transaction time range %d-%d", b.Body.TimestampMS, tx.Body.MinTimeMS, tx.Body.MaxTimeMS)
		}
		if tx.Body.MinTimeMS > 0 && b.Body.TimestampMS > 0 && b.Body.TimestampMS < tx.Body.MinTimeMS {
			return errors.WithDetailf(errUntimelyTransaction, "block timestamp %d, transaction time range %d-%d", b.Body.TimestampMS, tx.Body.MinTimeMS, tx.Body.MaxTimeMS)
		}

		vs2 := *vs
		vs2.tx = tx
		vs2.entryID = tx.ID

		err := tx.checkValid(&vs2)
		if err != nil {
			return errors.Wrapf(err, "checking validity of transaction %d of %d", i, len(b.Transactions))
		}
	}

	txRoot, err := MerkleRoot(b.Transactions)
	if err != nil {
		return errors.Wrap(err, "computing transaction merkle root")
	}

	if txRoot != b.Body.TransactionsRoot {
		return errors.WithDetailf(errMismatchedMerkleRoot, "computed %x, current block wants %x", txRoot[:], b.Body.TransactionsRoot[:])
	}

	return nil
}
