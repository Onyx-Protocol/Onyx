package tx

import (
	"fmt"
	"log"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"

	"github.com/davecgh/go-spew/spew"
)

func init() {
	bc.TxHashesFunc = TxHashes
}

// TxHashes returns all hashes needed for validation and state updates.
func TxHashes(oldTx *bc.TxData) (hashes *bc.TxHashes, err error) {
	txid, header, entries, err := mapTx(oldTx)
	if err != nil {
		return nil, errors.Wrap(err, "mapping old transaction to new")
	}

	hashes = new(bc.TxHashes)
	hashes.ID = bc.Hash(txid)

	// OutputIDs
	log.Println(spew.Sdump(header))
	for _, resultHash := range header.body.Results {
		result := entries[resultHash]
		if _, ok := result.(*output); ok {
			hashes.OutputIDs = append(hashes.OutputIDs, bc.Hash(resultHash))
		}
	}

	var txRefDataHash bc.Hash
	if header.body.Data == (entryRef{}) {
		// no data entry
		txRefDataHash = bc.EmptyStringHash
	} else {
		dEntry, ok := entries[header.body.Data]
		if !ok {
			return nil, fmt.Errorf("header refers to nonexistent data entry")
		}
		d, ok := dEntry.(*data)
		if !ok {
			return nil, fmt.Errorf("header refers to %s entry, should be data", dEntry.Type())
		}
		txRefDataHash = d.body
	}

	hashes.VMContexts = make([]*bc.VMContext, len(oldTx.Inputs))

	for entryID, ent := range entries {
		switch ent := ent.(type) {
		case *nonce:
			// TODO: check time range is within network-defined limits
			trID := ent.body.TimeRange
			trEntry, ok := entries[trID]
			if !ok {
				return nil, fmt.Errorf("nonce entry refers to nonexistent timerange entry")
			}
			tr, ok := trEntry.(*timeRange)
			if !ok {
				return nil, fmt.Errorf("nonce entry refers to %s entry, should be timerange", trEntry.Type())
			}
			iss := struct {
				ID           bc.Hash
				ExpirationMS uint64
			}{bc.Hash(entryID), tr.body.MaxTimeMS}
			hashes.Issuances = append(hashes.Issuances, iss)

		case *issuance:
			vmc := newVMContext(bc.Hash(entryID), hashes.ID, txRefDataHash)
			vmc.RefDataHash = bc.Hash(ent.body.Data)
			vmc.NonceID = (*bc.Hash)(&ent.body.Anchor)
			hashes.VMContexts[ent.Ordinal()] = vmc

		case *spend:
			vmc := newVMContext(bc.Hash(entryID), hashes.ID, txRefDataHash)
			vmc.RefDataHash = bc.Hash(ent.body.Data)
			vmc.OutputID = (*bc.Hash)(&ent.body.SpentOutput)
			hashes.VMContexts[ent.Ordinal()] = vmc
		}
	}

	return hashes, nil
}

// populates the common fields of a VMContext for an Entry, regardless of whether
// that Entry is a Spend or an Issuance
func newVMContext(entryID, txid, txRefDataHash bc.Hash) *bc.VMContext {
	vmc := new(bc.VMContext)

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
