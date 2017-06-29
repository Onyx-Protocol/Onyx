package validation

import (
	"chain/crypto/ed25519"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/bc/bcvm"
	"chain/protocol/txvm"
)

var (
	errBadTimeRange          = errors.New("bad time range")
	errEmptyResults          = errors.New("transaction has no results")
	errMismatchedAssetID     = errors.New("mismatched asset id")
	errMismatchedBlock       = errors.New("mismatched block")
	errMismatchedMerkleRoot  = errors.New("mismatched merkle root")
	errMismatchedPosition    = errors.New("mismatched value source/dest positions")
	errMismatchedReference   = errors.New("mismatched reference")
	errMismatchedValue       = errors.New("mismatched value")
	errMisorderedBlockHeight = errors.New("misordered block height")
	errMisorderedBlockTime   = errors.New("misordered block time")
	errMissingField          = errors.New("missing required field")
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

// ValidateBlockSig runs the consensus program prog on b.
func ValidateBlockSig(b *bcvm.Block, pubkeys [][]byte, quorum uint32) error {
	sigs := b.Witness
	if uint32(len(sigs)) < quorum {
		return errors.New("too few signatures")
	}
	sigs = sigs[:quorum]
	hash := b.HashForSig().Bytes()
	for len(sigs) > 0 && len(pubkeys) > 0 {
		if ed25519.Verify(pubkeys[0], hash, sigs[0]) {
			sigs = sigs[1:]
		}
		pubkeys = pubkeys[1:]
	}
	if len(sigs) > 0 {
		return errors.New("invalid signatures")
	}
	return nil
}

// ValidateBlock validates a block and the transactions within.
// It does not run the consensus program; for that, see ValidateBlockSig.
func ValidateBlock(b, prev *bcvm.Block) error {
	if b.Height > 1 {
		if prev == nil {
			return errors.WithDetailf(errNoPrevBlock, "height %d", b.Height)
		}
		err := validateBlockAgainstPrev(b, prev)
		if err != nil {
			return err
		}
	}

	var ids []bc.Hash

	for _, tx := range b.Transactions {
		deserialized, err := bcvm.NewTx(tx)
		if err != nil {
			return errors.Wrap(err)
		}

		for _, tc := range deserialized.TimeConstraints {
			if (tc.Type == "min" && b.TimestampMS < uint64(tc.Time)) || (tc.Type == "max" && b.TimestampMS > uint64(tc.Time)) {
				return errors.New("tx invalid due to timestamp")
			}
		}

		ids = append(ids, deserialized.ID)
	}

	txRoot, err := bcvm.MerkleRoot(ids)
	if err != nil {
		return errors.Wrap(err, "computing transaction merkle root")
	}

	if txRoot != b.TransactionsMerkleRoot {
		return errors.WithDetailf(errMismatchedMerkleRoot, "computed %x, current block wants %x", txRoot.Bytes(), b.TransactionsMerkleRoot.Bytes())
	}

	return nil
}

func validateBlockAgainstPrev(b, prev *bcvm.Block) error {
	if b.Version < prev.Version {
		return errors.WithDetailf(errVersionRegression, "previous block verson %d, current block version %d", prev.Version, b.Version)
	}
	if b.Height != prev.Height+1 {
		return errors.WithDetailf(errMisorderedBlockHeight, "previous block height %d, current block height %d", prev.Height, b.Height)
	}
	prevHash := prev.Hash()
	if prevHash != b.PreviousBlockHash {
		return errors.WithDetailf(errMismatchedBlock, "previous block ID %x, current block wants %x", prevHash.Bytes(), b.PreviousBlockHash.Bytes())
	}
	if b.TimestampMS <= prev.TimestampMS {
		return errors.WithDetailf(errMisorderedBlockTime, "previous block time %d, current block time %d", prev.TimestampMS, b.TimestampMS)
	}
	return nil
}

// ValidateTx validates a transaction.
func ValidateTx(tx []byte) error {
	_, ok := txvm.Validate(tx)
	if !ok {
		return errors.New("invalid tx")
	}
	return nil
}
