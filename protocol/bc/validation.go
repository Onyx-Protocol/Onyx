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
