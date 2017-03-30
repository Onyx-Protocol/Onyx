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
	IssuanceIDs    []Hash
	SpentOutputIDs []Hash
	OutputIDs      []Hash
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
		issuanceIDs    = make(map[Hash]bool)
		spentOutputIDs = make(map[Hash]bool)
		outputIDs      = make(map[Hash]bool)
	)

	for id, e := range entries {
		switch e := e.(type) {
		case *Issuance:
			issuanceIDs[id] = true
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

	for id := range issuanceIDs {
		txEntries.IssuanceIDs = append(txEntries.IssuanceIDs, id)
	}
	for id := range spentOutputIDs {
		txEntries.SpentOutputIDs = append(txEntries.SpentOutputIDs, id)
	}
	for id := range outputIDs {
		txEntries.OutputIDs = append(txEntries.OutputIDs, id)
	}

	return txEntries, nil
}
