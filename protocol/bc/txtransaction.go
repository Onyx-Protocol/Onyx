package bc

import (
	"fmt"

	"chain/crypto/sha3pool"
	"chain/errors"
)

// TxEntries is a wrapper for the entries-based representation of a
// transaction.  When we no longer need the legacy Tx and TxData
// types, this will be renamed Tx.
type TxEntries struct {
	*TxHeader
	ID         Hash
	Entries    map[Hash]Entry
	TxInputs   []Entry // 1:1 correspondence with TxData.Inputs
	TxInputIDs []Hash  // 1:1 correspondence with TxData.Inputs

	// IDs of reachable entries of various kinds to speed up Apply
	NonceIDs       []Hash
	SpentOutputIDs []Hash
	OutputIDs      []Hash
}

func (tx *TxEntries) SigHash(n uint32) (hash Hash) {
	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	tx.TxInputIDs[n].WriteTo(hasher)
	tx.ID.WriteTo(hasher)
	hash.ReadFrom(hasher)
	return hash
}

// ComputeOutputID assembles an output entry given a spend commitment
// and computes and returns its corresponding entry ID.
func ComputeOutputID(sc *SpendCommitment) (h Hash, err error) {
	defer func() {
		if r, ok := recover().(error); ok {
			err = r
		}
	}()
	src := &ValueSource{
		Ref:      &sc.SourceID,
		Value:    &sc.AssetAmount,
		Position: sc.SourcePosition,
	}
	o := NewOutput(src, &Program{VmVersion: sc.VMVersion, Code: sc.ControlProgram}, &sc.RefDataHash, 0)

	h = EntryID(o)
	return h, nil
}

// MapTx converts a legacy TxData object into its entries-based
// representation.
func MapTx(oldTx *TxData) (txEntries *TxEntries, err error) {
	defer func() {
		if r, ok := recover().(error); ok {
			err = r
		}
	}()

	txid, header, entries, err := mapTx(oldTx)
	if err != nil {
		return nil, errors.Wrap(err, "mapping old transaction to new")
	}

	txEntries = &TxEntries{
		TxHeader:   header,
		ID:         txid,
		Entries:    entries,
		TxInputs:   make([]Entry, len(oldTx.Inputs)),
		TxInputIDs: make([]Hash, len(oldTx.Inputs)),
	}

	var (
		nonceIDs       = make(map[Hash]bool)
		spentOutputIDs = make(map[Hash]bool)
		outputIDs      = make(map[Hash]bool)
	)

	for id, e := range entries {
		var ord uint64
		switch e := e.(type) {
		case *Issuance:
			anchor, ok := entries[*e.Body.AnchorId]
			if !ok {
				return nil, fmt.Errorf("entry for anchor ID %x not found", e.Body.AnchorId.Bytes())
			}
			if _, ok := anchor.(*Nonce); ok {
				nonceIDs[*e.Body.AnchorId] = true
			}
			ord = e.Ordinal
			// resume below after the switch

		case *Spend:
			spentOutputIDs[*e.Body.SpentOutputId] = true
			ord = e.Ordinal
			// resume below after the switch

		case *Output:
			outputIDs[id] = true
			continue

		default:
			continue
		}
		if ord >= uint64(len(oldTx.Inputs)) {
			return nil, fmt.Errorf("%T entry has out-of-range ordinal %d", e, ord)
		}
		txEntries.TxInputs[ord] = e
		txEntries.TxInputIDs[ord] = id
	}

	for id := range nonceIDs {
		txEntries.NonceIDs = append(txEntries.NonceIDs, id)
	}
	for id := range spentOutputIDs {
		txEntries.SpentOutputIDs = append(txEntries.SpentOutputIDs, id)
	}
	for id := range outputIDs {
		txEntries.OutputIDs = append(txEntries.OutputIDs, id)
	}

	return txEntries, nil
}

// Convenience routines for accessing entries of specific types by ID.

func (tx *TxEntries) TimeRange(id Hash) (*TimeRange, error) {
	e, ok := tx.Entries[id]
	if !ok {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	tr, ok := e.(*TimeRange)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return tr, nil
}

func (tx *TxEntries) Output(id Hash) (*Output, error) {
	e, ok := tx.Entries[id]
	if !ok {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	o, ok := e.(*Output)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return o, nil
}
