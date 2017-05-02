package bc

import (
	"chain/crypto/sha3pool"
	"chain/errors"
)

// Tx is a wrapper for the entries-based representation of a transaction.
type Tx struct {
	*TxHeader
	ID       Hash
	Entries  map[Hash]Entry
	InputIDs []Hash // 1:1 correspondence with TxData.Inputs

	// IDs of reachable entries of various kinds
	NonceIDs       []Hash
	SpentOutputIDs []Hash
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
	if !ok || e == nil {
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
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	o, ok := e.(*Output)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return o, nil
}

func (tx *Tx) Spend(id Hash) (*Spend, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	sp, ok := e.(*Spend)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return sp, nil
}

func (tx *Tx) Issuance(id Hash) (*Issuance, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	iss, ok := e.(*Issuance)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return iss, nil
}

func (tx *Tx) Nonce(id Hash) (*Nonce, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	nonce, ok := e.(*Nonce)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return nonce, nil
}
