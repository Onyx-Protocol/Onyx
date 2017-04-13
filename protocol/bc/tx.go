package bc

import (
	"chain/crypto/sha3pool"
	"chain/errors"
)

// TxEntries is a wrapper for the entries-based representation of a
// transaction.  When we no longer need the legacy Tx and TxData
// types, this will be renamed Tx.
type Tx struct {
	*TxHeader
	ID       Hash
	Entries  map[Hash]Entry
	TxInputs []Entry // 1:1 correspondence with TxData.Inputs
	InputIDs []Hash  // 1:1 correspondence with TxData.Inputs

	// IDs of reachable entries of various kinds to speed up Apply
	NonceIDs       []Hash
	SpentOutputIDs []Hash
	OutputIDs      []Hash
}

func (tx *Tx) SigHash(n uint32) (hash Hash) {
	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	tx.InputIDs[n].WriteTo(hasher)
	tx.ID.WriteTo(hasher)
	hash.ReadFrom(hasher)
	return hash
}

// Convenience routines for accessing entries of specific types by ID.

var (
	ErrEntryType    = errors.New("invalid entry type")
	ErrMissingEntry = errors.New("missing entry")
)

func (tx *Tx) TimeRange(id Hash) (*TimeRange, error) {
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

func (tx *Tx) Output(id Hash) (*Output, error) {
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
