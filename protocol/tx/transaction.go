package tx

import (
	"chain/crypto/sha3pool"
	"chain/protocol/bc"
)

// The data needed for validation and state updates.
type TxHashes struct {
	ID        entryRef
	OutputIDs []entryRef // each OutputID is also the corresponding UnspentID
	Issuances []struct {
		ID           entryRef
		ExpirationMS uint64
	}
	VMContexts []*VMContext // one per old-style Input
}

type VMContext struct {
	TxRefDataHash bc.Hash
	RefDataHash   bc.Hash
	TxSigHash     bc.Hash
	OutputID      *bc.Hash
	EntryID       bc.Hash
	AnchorID      *bc.Hash
}

// HashTx returns all hashes needed for validation and state updates.
func HashTx(oldTx *bc.TxData) (hashes *TxHashes, err error) {
	header, entries, err := mapTx(oldTx)
	if err != nil {
		return nil, err
	}

	hashes = new(TxHashes)

	// ID
	hashes.ID, err = entryID(header)
	if err != nil {
		return nil, err
	}

	// OutputIDs
	for _, resultHash := range header.body.Results {
		result := entries[resultHash]
		if _, ok := result.(*output); ok {
			hashes.OutputIDs = append(hashes.OutputIDs, resultHash)
		}
	}

	var txRefDataHash bc.Hash // xxx calculate this for the tx

	for entryID, ent := range entries {
		switch ent := ent.(type) {
		case *anchor:
			// xxx check time range is within network-defined limits
			trID := ent.body.TimeRange
			trEntry := entries[trID].(*timeRange) // xxx avoid panics here
			iss := struct {
				ID           entryRef
				ExpirationMS uint64
			}{entryID, trEntry.body.MaxTimeMS}
			hashes.Issuances = append(hashes.Issuances, iss)

		case *issuance:
			vmc := newVMContext(bc.Hash(entryID), bc.Hash(hashes.ID), txRefDataHash)
			vmc.RefDataHash = bc.Hash(ent.body.Data) // xxx should this be the id of the data entry? or the hash of the data that's _in_ the data entry?
			vmc.AnchorID = (*bc.Hash)(&ent.body.Anchor)
			hashes.VMContexts = append(hashes.VMContexts, vmc)

		case *spend:
			vmc := newVMContext(bc.Hash(entryID), bc.Hash(hashes.ID), txRefDataHash)
			vmc.RefDataHash = bc.Hash(ent.body.Reference)
			vmc.OutputID = (*bc.Hash)(&ent.body.SpentOutput)
			hashes.VMContexts = append(hashes.VMContexts, vmc)
		}
	}

	return hashes, nil
}

// populates the common fields of a VMContext for an Entry, regardless of whether
// that Entry is a Spend or an Issuance
func newVMContext(entryID, txid, txRefDataHash bc.Hash) *VMContext {
	vmc := new(VMContext)

	// TxRefDataHash
	vmc.TxRefDataHash = txRefDataHash

	// EntryID
	vmc.EntryID = entryID

	// TxSigHash
	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)
	hasher.Write(entryID[:])
	hasher.Write(txid[:])
	hasher.Read(vmc.TxSigHash[:])

	return vmc
}
