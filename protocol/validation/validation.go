package validation

import (
	"fmt"

	"chain/errors"
	"chain/math/checked"
	"chain/protocol/bc"
	"chain/protocol/vm"
)

// validationState contains the context that must propagate through
// the transaction graph when validating entries.
type validationState struct {
	// The ID of the blockchain
	blockchainID bc.Hash

	// The enclosing transaction object
	tx *bc.Tx

	// The ID of the nearest enclosing entry
	entryID bc.Hash

	// The source position, for validating ValueSources
	sourcePos uint64

	// The destination position, for validating ValueDestinations
	destPos uint64

	// Memoized per-entry validation results
	cache map[bc.Hash]error
}

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

func checkValid(vs *validationState, e bc.Entry) (err error) {
	entryID := bc.EntryID(e)
	var ok bool
	if err, ok = vs.cache[entryID]; ok {
		return err
	}

	defer func() {
		vs.cache[entryID] = err
	}()

	switch e := e.(type) {
	case *bc.TxHeader:
		// This does only part of the work of validating a tx header. The
		// block-related parts of tx validation are in ValidateBlock.
		if e.MaxTimeMs > 0 {
			if e.MaxTimeMs < e.MinTimeMs {
				return errors.WithDetailf(errBadTimeRange, "min time %d, max time %d", e.MinTimeMs, e.MaxTimeMs)
			}
		}

		for i, resID := range e.ResultIds {
			resultEntry := vs.tx.Entries[*resID]
			vs2 := *vs
			vs2.entryID = *resID
			err = checkValid(&vs2, resultEntry)
			if err != nil {
				return errors.Wrapf(err, "checking result %d", i)
			}
		}

		if e.Version == 1 {
			if len(e.ResultIds) == 0 {
				return errEmptyResults
			}

			if e.ExtHash != nil && !e.ExtHash.IsZero() {
				return errNonemptyExtHash
			}
		}

	case *bc.Mux:
		err = vm.Verify(NewTxVMContext(vs.tx, e, e.Program, e.WitnessArguments))
		if err != nil {
			return errors.Wrap(err, "checking mux program")
		}

		for i, src := range e.Sources {
			vs2 := *vs
			vs2.sourcePos = uint64(i)
			err = checkValidSrc(&vs2, src)
			if err != nil {
				return errors.Wrapf(err, "checking mux source %d", i)
			}
		}
		for i, dest := range e.WitnessDestinations {
			vs2 := *vs
			vs2.destPos = uint64(i)
			err = checkValidDest(&vs2, dest)
			if err != nil {
				return errors.Wrapf(err, "checking mux destination %d", i)
			}
		}

		parity := make(map[bc.AssetID]int64)
		for i, src := range e.Sources {
			sum, ok := checked.AddInt64(parity[*src.Value.AssetId], int64(src.Value.Amount))
			if !ok {
				return errors.WithDetailf(errOverflow, "adding %d units of asset %x from mux source %d to total %d overflows int64", src.Value.Amount, src.Value.AssetId.Bytes(), i, parity[*src.Value.AssetId])
			}
			parity[*src.Value.AssetId] = sum
		}

		for i, dest := range e.WitnessDestinations {
			sum, ok := parity[*dest.Value.AssetId]
			if !ok {
				return errors.WithDetailf(errNoSource, "mux destination %d, asset %x, has no corresponding source", i, dest.Value.AssetId.Bytes())
			}

			diff, ok := checked.SubInt64(sum, int64(dest.Value.Amount))
			if !ok {
				return errors.WithDetailf(errOverflow, "subtracting %d units of asset %x from mux destination %d from total %d underflows int64", dest.Value.Amount, dest.Value.AssetId.Bytes(), i, sum)
			}
			parity[*dest.Value.AssetId] = diff
		}

		for assetID, amount := range parity {
			if amount != 0 {
				return errors.WithDetailf(errUnbalanced, "asset %x sources - destinations = %d (should be 0)", assetID.Bytes(), amount)
			}
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	case *bc.Nonce:
		err = vm.Verify(NewTxVMContext(vs.tx, e, e.Program, e.WitnessArguments))
		if err != nil {
			return errors.Wrap(err, "checking nonce program")
		}
		tr, err := vs.tx.TimeRange(*e.TimeRangeId)
		if err != nil {
			return errors.Wrap(err, "getting nonce timerange")
		}
		vs2 := *vs
		vs2.entryID = *e.TimeRangeId
		err = checkValid(&vs2, tr)
		if err != nil {
			return errors.Wrap(err, "checking nonce timerange")
		}

		if tr.MinTimeMs == 0 || tr.MaxTimeMs == 0 {
			return errZeroTime
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	case *bc.Output:
		vs2 := *vs
		vs2.sourcePos = 0
		err = checkValidSrc(&vs2, e.Source)
		if err != nil {
			return errors.Wrap(err, "checking output source")
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	case *bc.Retirement:
		vs2 := *vs
		vs2.sourcePos = 0
		err = checkValidSrc(&vs2, e.Source)
		if err != nil {
			return errors.Wrap(err, "checking retirement source")
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	case *bc.TimeRange:
		if e.MinTimeMs > vs.tx.MinTimeMs {
			return errBadTimeRange
		}
		if e.MaxTimeMs > 0 && e.MaxTimeMs < vs.tx.MaxTimeMs {
			return errBadTimeRange
		}
		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	case *bc.Issuance:
		if *e.WitnessAssetDefinition.InitialBlockId != vs.blockchainID {
			return errors.WithDetailf(errWrongBlockchain, "current blockchain %x, asset defined on blockchain %x", vs.blockchainID.Bytes(), e.WitnessAssetDefinition.InitialBlockId.Bytes())
		}

		computedAssetID := e.WitnessAssetDefinition.ComputeAssetID()
		if computedAssetID != *e.Value.AssetId {
			return errors.WithDetailf(errMismatchedAssetID, "asset ID is %x, issuance wants %x", computedAssetID.Bytes(), e.Value.AssetId.Bytes())
		}

		anchor, ok := vs.tx.Entries[*e.AnchorId]
		if !ok {
			return errors.Wrapf(bc.ErrMissingEntry, "entry for issuance anchor %x not found", e.AnchorId.Bytes())
		}

		err = vm.Verify(NewTxVMContext(vs.tx, e, e.WitnessAssetDefinition.IssuanceProgram, e.WitnessArguments))
		if err != nil {
			return errors.Wrap(err, "checking issuance program")
		}

		var anchored *bc.Hash
		switch a := anchor.(type) {
		case *bc.Nonce:
			anchored = a.WitnessAnchoredId

		case *bc.Spend:
			anchored = a.WitnessAnchoredId

		case *bc.Issuance:
			anchored = a.WitnessAnchoredId

		default:
			return errors.WithDetailf(bc.ErrEntryType, "issuance anchor has type %T, should be nonce, spend, or issuance", anchor)
		}

		if *anchored != vs.entryID {
			return errors.WithDetailf(errMismatchedReference, "issuance %x anchor is for %x", vs.entryID.Bytes(), anchored.Bytes())
		}

		anchorVS := *vs
		anchorVS.entryID = *e.AnchorId
		err = checkValid(&anchorVS, anchor)
		if err != nil {
			return errors.Wrap(err, "checking issuance anchor")
		}

		destVS := *vs
		destVS.destPos = 0
		err = checkValidDest(&destVS, e.WitnessDestination)
		if err != nil {
			return errors.Wrap(err, "checking issuance destination")
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	case *bc.Spend:
		if e.SpentOutputId == nil {
			return errors.Wrap(errMissingField, "spend without spent output ID")
		}
		spentOutput, err := vs.tx.Output(*e.SpentOutputId)
		if err != nil {
			return errors.Wrap(err, "getting spend prevout")
		}
		err = vm.Verify(NewTxVMContext(vs.tx, e, spentOutput.ControlProgram, e.WitnessArguments))
		if err != nil {
			return errors.Wrap(err, "checking control program")
		}

		eq, err := spentOutput.Source.Value.Equal(e.WitnessDestination.Value)
		if err != nil {
			return err
		}
		if !eq {
			return errors.WithDetailf(
				errMismatchedValue,
				"previous output is for %d unit(s) of %x, spend wants %d unit(s) of %x",
				spentOutput.Source.Value.Amount,
				spentOutput.Source.Value.AssetId.Bytes(),
				e.WitnessDestination.Value.Amount,
				e.WitnessDestination.Value.AssetId.Bytes(),
			)
		}

		vs2 := *vs
		vs2.destPos = 0
		err = checkValidDest(&vs2, e.WitnessDestination)
		if err != nil {
			return errors.Wrap(err, "checking spend destination")
		}

		if vs.tx.Version == 1 && e.ExtHash != nil && !e.ExtHash.IsZero() {
			return errNonemptyExtHash
		}

	default:
		return fmt.Errorf("entry has unexpected type %T", e)
	}

	return nil
}

func checkValidBlockHeader(bh *bc.BlockHeader) error {
	if bh.Version == 1 && bh.ExtHash != nil && !bh.ExtHash.IsZero() {
		return errNonemptyExtHash
	}
	return nil
}

func checkValidSrc(vstate *validationState, vs *bc.ValueSource) error {
	if vs == nil {
		return errors.Wrap(errMissingField, "empty value source")
	}
	if vs.Ref == nil {
		return errors.Wrap(errMissingField, "missing ref on value source")
	}
	if vs.Value == nil || vs.Value.AssetId == nil {
		return errors.Wrap(errMissingField, "missing value on value source")
	}

	e, ok := vstate.tx.Entries[*vs.Ref]
	if !ok {
		return errors.Wrapf(bc.ErrMissingEntry, "entry for value source %x not found", vs.Ref.Bytes())
	}
	vstate2 := *vstate
	vstate2.entryID = *vs.Ref
	err := checkValid(&vstate2, e)
	if err != nil {
		return errors.Wrap(err, "checking value source")
	}

	var dest *bc.ValueDestination
	switch ref := e.(type) {
	case *bc.Issuance:
		if vs.Position != 0 {
			return errors.Wrapf(errPosition, "invalid position %d for issuance source", vs.Position)
		}
		dest = ref.WitnessDestination

	case *bc.Spend:
		if vs.Position != 0 {
			return errors.Wrapf(errPosition, "invalid position %d for spend source", vs.Position)
		}
		dest = ref.WitnessDestination

	case *bc.Mux:
		if vs.Position >= uint64(len(ref.WitnessDestinations)) {
			return errors.Wrapf(errPosition, "invalid position %d for %d-destination mux source", vs.Position, len(ref.WitnessDestinations))
		}
		dest = ref.WitnessDestinations[vs.Position]

	default:
		return errors.Wrapf(bc.ErrEntryType, "value source is %T, should be issuance, spend, or mux", e)
	}

	if dest.Ref == nil || *dest.Ref != vstate.entryID {
		return errors.Wrapf(errMismatchedReference, "value source for %x has disagreeing destination %x", vstate.entryID.Bytes(), dest.Ref.Bytes())
	}

	if dest.Position != vstate.sourcePos {
		return errors.Wrapf(errMismatchedPosition, "value source position %d disagrees with %d", dest.Position, vstate.sourcePos)
	}

	eq, err := dest.Value.Equal(vs.Value)
	if err != nil {
		return errors.Sub(errMissingField, err)
	}
	if !eq {
		return errors.Wrapf(errMismatchedValue, "source value %v disagrees with %v", dest.Value, vs.Value)
	}

	return nil
}

func checkValidDest(vs *validationState, vd *bc.ValueDestination) error {
	if vd == nil {
		return errors.Wrap(errMissingField, "empty value destination")
	}
	if vd.Ref == nil {
		return errors.Wrap(errMissingField, "missing ref on value destination")
	}
	if vd.Value == nil || vd.Value.AssetId == nil {
		return errors.Wrap(errMissingField, "missing value on value source")
	}

	e, ok := vs.tx.Entries[*vd.Ref]
	if !ok {
		return errors.Wrapf(bc.ErrMissingEntry, "entry for value destination %x not found", vd.Ref.Bytes())
	}
	var src *bc.ValueSource
	switch ref := e.(type) {
	case *bc.Output:
		if vd.Position != 0 {
			return errors.Wrapf(errPosition, "invalid position %d for output destination", vd.Position)
		}
		src = ref.Source

	case *bc.Retirement:
		if vd.Position != 0 {
			return errors.Wrapf(errPosition, "invalid position %d for retirement destination", vd.Position)
		}
		src = ref.Source

	case *bc.Mux:
		if vd.Position >= uint64(len(ref.Sources)) {
			return errors.Wrapf(errPosition, "invalid position %d for %d-source mux destination", vd.Position, len(ref.Sources))
		}
		src = ref.Sources[vd.Position]

	default:
		return errors.Wrapf(bc.ErrEntryType, "value destination is %T, should be output, retirement, or mux", e)
	}

	if src.Ref == nil || *src.Ref != vs.entryID {
		return errors.Wrapf(errMismatchedReference, "value destination for %x has disagreeing source %x", vs.entryID.Bytes(), src.Ref.Bytes())
	}

	if src.Position != vs.destPos {
		return errors.Wrapf(errMismatchedPosition, "value destination position %d disagrees with %d", src.Position, vs.destPos)
	}

	eq, err := src.Value.Equal(vd.Value)
	if err != nil {
		return errors.Sub(errMissingField, err)
	}
	if !eq {
		return errors.Wrapf(errMismatchedValue, "destination value %v disagrees with %v", src.Value, vd.Value)
	}

	return nil
}

// ValidateBlockSig runs the consensus program prog on b.
func ValidateBlockSig(b *bc.Block, prog []byte) error {
	vmContext := newBlockVMContext(b, prog, b.WitnessArguments)
	err := vm.Verify(vmContext)
	return errors.Wrap(err, "evaluating previous block's next consensus program")
}

// ValidateBlock validates a block and the transactions within.
// It does not run the consensus program; for that, see ValidateBlockSig.
func ValidateBlock(b, prev *bc.Block, initialBlockID bc.Hash, validateTx func(*bc.Tx) error) error {
	if b.Height > 1 {
		if prev == nil {
			return errors.WithDetailf(errNoPrevBlock, "height %d", b.Height)
		}
		err := validateBlockAgainstPrev(b, prev)
		if err != nil {
			return err
		}
	}

	err := checkValidBlockHeader(b.BlockHeader)
	if err != nil {
		return errors.Wrap(err, "checking block header")
	}

	for i, tx := range b.Transactions {
		if b.Version == 1 && tx.Version != 1 {
			return errors.WithDetailf(errTxVersion, "block version %d, transaction version %d", b.Version, tx.Version)
		}
		if tx.MaxTimeMs > 0 && b.TimestampMs > tx.MaxTimeMs {
			return errors.WithDetailf(errUntimelyTransaction, "block timestamp %d, transaction time range %d-%d", b.TimestampMs, tx.MinTimeMs, tx.MaxTimeMs)
		}
		if tx.MinTimeMs > 0 && b.TimestampMs > 0 && b.TimestampMs < tx.MinTimeMs {
			return errors.WithDetailf(errUntimelyTransaction, "block timestamp %d, transaction time range %d-%d", b.TimestampMs, tx.MinTimeMs, tx.MaxTimeMs)
		}

		err = validateTx(tx)
		if err != nil {
			return errors.Wrapf(err, "validity of transaction %d of %d", i, len(b.Transactions))
		}
	}

	txRoot, err := bc.MerkleRoot(b.Transactions)
	if err != nil {
		return errors.Wrap(err, "computing transaction merkle root")
	}

	if txRoot != *b.TransactionsRoot {
		return errors.WithDetailf(errMismatchedMerkleRoot, "computed %x, current block wants %x", txRoot.Bytes(), b.TransactionsRoot.Bytes())
	}

	return nil
}

func validateBlockAgainstPrev(b, prev *bc.Block) error {
	if b.Version < prev.Version {
		return errors.WithDetailf(errVersionRegression, "previous block verson %d, current block version %d", prev.Version, b.Version)
	}
	if b.Height != prev.Height+1 {
		return errors.WithDetailf(errMisorderedBlockHeight, "previous block height %d, current block height %d", prev.Height, b.Height)
	}
	if prev.ID != *b.PreviousBlockId {
		return errors.WithDetailf(errMismatchedBlock, "previous block ID %x, current block wants %x", prev.ID.Bytes(), b.PreviousBlockId.Bytes())
	}
	if b.TimestampMs <= prev.TimestampMs {
		return errors.WithDetailf(errMisorderedBlockTime, "previous block time %d, current block time %d", prev.TimestampMs, b.TimestampMs)
	}
	return nil
}

// ValidateTx validates a transaction.
func ValidateTx(tx *bc.Tx, initialBlockID bc.Hash) error {
	vs := &validationState{
		blockchainID: initialBlockID,
		tx:           tx,
		entryID:      tx.ID,

		cache: make(map[bc.Hash]error),
	}
	return checkValid(vs, tx.TxHeader)
}
