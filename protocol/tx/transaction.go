package tx

import (
	"chain/crypto/sha3pool"
	"chain/protocol/bc"
)

// The data needed for validation and state updates.
type TxHashes struct {
	ID        bc.Hash
	OutputIDs []bc.Hash // each OutputID is also the corresponding UnspentID
	Issuances []struct {
		ID           bc.Hash
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
	headerEntry, entries, err := mapTx(oldTx)
	if err != nil {
		return nil, err
	}

	hashes = new(TxHashes)

	// ID
	txid, err := entryID(headerEntry)
	if err != nil {
		return nil, err
	}
	hashes.ID = bc.Hash(txid)

	header := headerEntry.body.(*header)

	// OutputIDs
	for _, resultHash := range header.Results {
		result := entries[resultHash]
		if _, ok := result.body.(*output); ok {
			hashes.OutputIDs = append(hashes.OutputIDs, bc.Hash(resultHash))
		}
	}

	var txRefDataHash bc.Hash // xxx calculate this for the tx

	for entryID, ent := range entries {
		switch body := ent.body.(type) {
		case *anchor:
			// xxx check time range is within network-defined limits
			trID := body.TimeRange
			trBody := entries[trID].body.(*timeRange) // xxx avoid panics here
			iss := struct {
				ID           bc.Hash
				ExpirationMS uint64
			}{bc.Hash(entryID), trBody.MaxTimeMS}
			hashes.Issuances = append(hashes.Issuances, iss)

		case *issuance:
			vmc := newVMContext(bc.Hash(entryID), hashes.ID, txRefDataHash)
			vmc.RefDataHash = bc.Hash(body.Data) // xxx should this be the id of the data entry? or the hash of the data that's _in_ the data entry?
			vmc.AnchorID = (*bc.Hash)(&body.Anchor)
			hashes.VMContexts = append(hashes.VMContexts, vmc)

		case *spend:
			vmc := newVMContext(bc.Hash(entryID), hashes.ID, txRefDataHash)
			vmc.RefDataHash = bc.Hash(body.Reference)
			vmc.OutputID = (*bc.Hash)(&body.SpentOutput)
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
