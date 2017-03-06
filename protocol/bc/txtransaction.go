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
	TxInputs   []Entry // 1:1 correspondence with TxData.Inputs
	TxInputIDs []Hash  // 1:1 correspondence with TxData.Inputs

	// IDs of reachable entries of various kinds to speed up Apply
	NonceIDs       []Hash
	SpentOutputIDs []Hash
	OutputIDs      []Hash
}

// ValidateTx validates a transaction.
func ValidateTx(tx *TxEntries, initialBlockID Hash) error {
	vs := &validationState{
		blockchainID: initialBlockID,
	}
	return tx.CheckValid(vs)
}

func (tx *TxEntries) CheckValid(vs *validationState) error {
	vs2 := *vs
	vs2.tx = tx
	vs2.entryID = tx.ID
	return tx.TxHeader.CheckValid(&vs2)
}

type BlockchainState interface {
	// AddNonce adds a nonce entry's ID and its expiration time T to the
	// state's nonce set.  It is an error for the nonce ID (with an
	// expiry >= T) to already be present.
	AddNonce(Hash, uint64) error

	// DeleteSpentOutput removes an output ID from the utxo set. It is
	// an error for the ID not to be present.
	DeleteSpentOutput(Hash) error

	// AddOutput adds an output ID to the utxo set.
	AddOutput(Hash) error
}

func (tx *TxEntries) Apply(state BlockchainState) error {
	for _, n := range tx.NonceIDs {
		err := state.AddNonce(n, tx.Body.MaxTimeMS)
		if err != nil {
			return err
		}
	}
	for _, s := range tx.SpentOutputIDs {
		err := state.DeleteSpentOutput(s)
		if err != nil {
			return err
		}
	}
	for _, o := range tx.OutputIDs {
		err := state.AddOutput(o)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tx *TxEntries) SigHash(n uint32) (hash Hash) {
	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	hasher.Write(tx.TxInputIDs[n][:])
	hasher.Write(tx.ID[:])

	hasher.Read(hash[:])
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
	src := ValueSource{
		Ref:      sc.SourceID,
		Value:    sc.AssetAmount,
		Position: sc.SourcePosition,
	}
	o := NewOutput(src, Program{VMVersion: sc.VMVersion, Code: sc.ControlProgram}, sc.RefDataHash, 0)

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
		TxInputs:   make([]Entry, len(oldTx.Inputs)),
		TxInputIDs: make([]Hash, len(oldTx.Inputs)),
	}

	var (
		nonceIDs       = make(map[Hash]bool)
		spentOutputIDs = make(map[Hash]bool)
		outputIDs      = make(map[Hash]bool)
	)

	for id, e := range entries {
		switch e := e.(type) {
		case *Issuance:
			if _, ok := e.Anchor.(*Nonce); ok {
				nonceIDs[e.Body.AnchorID] = true
			}
			// resume below after the switch

		case *Spend:
			spentOutputIDs[e.Body.SpentOutputID] = true
			// resume below after the switch

		case *Output:
			outputIDs[id] = true
			continue

		default:
			continue
		}
		ord := e.Ordinal()
		if ord < 0 || ord >= len(oldTx.Inputs) {
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
