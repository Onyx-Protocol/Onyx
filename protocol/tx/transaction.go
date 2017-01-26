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
	VMContexts []VMContext // one per old-style Input
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
func HashTx(oldTx *bc.TxData) (hashes *TxHashes, vmcs []*VMContext, err error) {
	header, entries, err := mapTx(oldTx)
	if err != nil {
		return nil, nil, err
	}

	hashes = new(TxHashes)

	// ID
	hashes.ID, err = entryID(header)
	if err != nil {
		return nil, nil, err
	}

	// OutputIDs
	for _, resultHash := range header.body.results {
		result := entries[resultHash]
		if _, ok := result.(*output); ok {
			hashes.OutputIDs = append(hashes.OutputIDs, resultHash)
		}
	}

	var txRefDataHash bc.Hash // TODO: calculate this for the tx

	for entryID, ent := range entries {
		switch ent := ent.(type) {
		case *anchor:
			// TODO: check time range is within network-defined limits
			trID := ent.body.timeRange
			trEntry := entries[trID].(*timeRange) // xxx avoid panics here
			iss := struct {
				ID           entryRef
				ExpirationMS uint64
			}{entryID, trEntry.body.maxTimeMS}
			hashes.Issuances = append(hashes.Issuances, iss)

		case *issuance:
			vmc := newVMContext(bc.Hash(entryID), bc.Hash(hashes.ID), txRefDataHash)
			vmc.RefDataHash = bc.Hash(ent.body.data) // xxx should this be the id of the data entry? or the hash of the data that's _in_ the data entry?
			vmc.AnchorID = (*bc.Hash)(&ent.body.anchor)
			vmcs = append(vmcs, vmc)

		case *spend:
			vmc := newVMContext(bc.Hash(entryID), bc.Hash(hashes.ID), txRefDataHash)
			vmc.RefDataHash = bc.Hash(ent.body.reference)
			vmc.OutputID = (*bc.Hash)(&ent.body.spentOutput)
			vmcs = append(vmcs, vmc)
		}
	}

	return hashes, vmcs, nil
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
