package bc

import (
	"github.com/golang/protobuf/proto"

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
	InputIDs []Hash // 1:1 correspondence with TxData.Inputs

	// IDs of reachable entries of various kinds
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

func (tx *Tx) Spend(id Hash) (*Spend, error) {
	e, ok := tx.Entries[id]
	if !ok {
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
	if !ok {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	iss, ok := e.(*Issuance)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return iss, nil
}

func (tx Tx) Proto() *ProtoTx {
	p := &ProtoTx{Header: tx.TxHeader}
	for _, e := range tx.Entries {
		switch e := e.(type) {
		case *Mux:
			p.Muxes = append(p.Muxes, e)
		case *Nonce:
			p.Nonces = append(p.Nonces, e)
		case *Output:
			p.Outputs = append(p.Outputs, e)
		case *Retirement:
			p.Retirements = append(p.Retirements, e)
		case *TimeRange:
			p.Timeranges = append(p.Timeranges, e)
		case *Issuance:
			p.Issuances = append(p.Issuances, e)
		case *Spend:
			p.Spends = append(p.Spends, e)
		}
	}
	return p
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (tx Tx) MarshalBinary() ([]byte, error) {
	return proto.Marshal(tx.Proto())
}

// MarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (tx *Tx) UnmarshalBinary(b []byte) error {
	var p ProtoTx
	err := proto.Unmarshal(b, &p)
	if err != nil {
		return err
	}
	tx.TxHeader = p.Header
	tx.ID = EntryID(tx.TxHeader)
	tx.Entries = make(map[Hash]Entry)
	tx.InputIDs = []Hash{}
	tx.NonceIDs = []Hash{}
	tx.SpentOutputIDs = []Hash{}
	tx.OutputIDs = []Hash{}
	for _, m := range p.Muxes {
		tx.Entries[EntryID(m)] = m
	}
	for _, n := range p.Nonces {
		id := EntryID(n)
		tx.Entries[id] = n
		tx.NonceIDs = append(tx.NonceIDs, id)
	}
	outputIDs := make(map[Hash]bool)
	for _, o := range p.Outputs {
		id := EntryID(o)
		tx.Entries[id] = o
		outputIDs[id] = true
	}
	for _, r := range p.Retirements {
		tx.Entries[EntryID(r)] = r
	}
	for _, tr := range p.Timeranges {
		tx.Entries[EntryID(tr)] = tr
	}
	for _, iss := range p.Issuances {
		tx.Entries[EntryID(iss)] = iss
	}
	for _, sp := range p.Spends {
		id := EntryID(sp)
		tx.Entries[id] = sp
		tx.SpentOutputIDs = append(tx.SpentOutputIDs, *sp.Body.SpentOutputId)
		outputIDs[id] = false
	}
	for id, ok := range outputIDs {
		if ok {
			tx.OutputIDs = append(tx.OutputIDs, id)
		}
	}
	// xxx populate inputids
	return nil
}
