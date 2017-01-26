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
func HashTx(oldTx *bc.TxData) (*TxHashes, error) {
	header, entries := mapTx(oldTx)
	hashes := new(TxHashes)

	// ID
	var err error
	hashes.ID, err = header.ID()
	if err != nil {
		return nil, err
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
			iss := struct {
				ID           bc.Hash
				ExpirationMS uint64
			}{entryID, ent.body.TimeRange.MaxTime}

			hashes.Issuances = append(hashes.Issuances, iss)
		case *issuance:
			vmc := newVMContext(entryID, txRefDataHash)
			vmc.RefDataHash = ent.body.data
			vmc.AnchorID = &ent.body.Anchor
		case *spend:
			vmc := newVMContext(entryID, txRefDataHash)
			vmc.RefDataHash = ent.body.data
			vmc.OutputID = &ent.body.SpentOutput
		}
	}

	return hashes, nil
}

// populates the common fields of a VMContext for an Entry, regardless of whether
// that Entry is a Spend or an Issuance
func newVMContext(entryID, txid, txRefDataHash bc.Hash) VMContext {
	vmc := new(VMContext)

	// TxRefDataHash
	vmc.TxRefDataHash = txRefDataHash

	// EntryID
	vmc.EntryID = entryID

	// TxSigHash
	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)
	hasher.Write(entryID)
	hasher.Write(txid)
	hasher.Read(vmc.TxSigHash[:])

	return vmc
}
