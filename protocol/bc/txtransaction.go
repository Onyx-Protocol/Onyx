package bc

import (
	"fmt"

	"chain/crypto/sha3pool"
	"chain/errors"
)

// ComputeOutputID assembles an output entry given a spend commitment
// and computes and returns its corresponding entry ID.
func ComputeOutputID(sc *SpendCommitment) (h Hash, err error) {
	defer func() {
		if r, ok := recover().(error); ok {
			err = r
		}
	}()
	o := newOutput(program{VMVersion: sc.VMVersion, Code: sc.ControlProgram}, sc.RefDataHash, 0)
	o.setSourceID(sc.SourceID, sc.AssetAmount, sc.SourcePosition)

	h = entryID(o)
	return h, nil
}

// TxHashes returns all hashes needed for validation and state updates.
func ComputeTxHashes(oldTx *TxData) (hashes *TxHashes, err error) {
	defer func() {
		if r, ok := recover().(error); ok {
			err = r
		}
	}()

	txid, header, entries, err := mapTx(oldTx)
	if err != nil {
		return nil, errors.Wrap(err, "mapping old transaction to new")
	}

	hashes = new(TxHashes)
	hashes.ID = txid

	// Results
	hashes.Results = make([]ResultInfo, len(header.body.Results))
	for i, resultHash := range header.body.Results {
		hashes.Results[i].ID = resultHash
		entry := entries[resultHash]
		if out, ok := entry.(*output); ok {
			hashes.Results[i].SourceID = out.body.Source.Ref
			hashes.Results[i].SourcePos = out.body.Source.Position
			hashes.Results[i].RefDataHash = out.body.Data
		}
	}

	hashes.VMContexts = make([]*VMContext, len(oldTx.Inputs))
	hashes.SpentOutputIDs = make([]Hash, len(oldTx.Inputs))

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
				ID           Hash
				ExpirationMS uint64
			}{entryID, tr.body.MaxTimeMS}
			hashes.Issuances = append(hashes.Issuances, iss)

		case *issuance:
			vmc := newVMContext(entryID, hashes.ID, header.body.Data, ent.body.Data)
			vmc.NonceID = (*Hash)(&ent.body.Anchor)
			hashes.VMContexts[ent.Ordinal()] = vmc

		case *spend:
			vmc := newVMContext(entryID, hashes.ID, header.body.Data, ent.body.Data)
			vmc.OutputID = (*Hash)(&ent.body.SpentOutput)
			hashes.VMContexts[ent.Ordinal()] = vmc
			hashes.SpentOutputIDs[ent.Ordinal()] = ent.body.SpentOutput
		}
	}

	return hashes, nil
}

// populates the common fields of a VMContext for an Entry, regardless of whether
// that Entry is a Spend or an Issuance
func newVMContext(entryID, txid, txData, inpData Hash) *VMContext {
	vmc := new(VMContext)

	// TxRefDataHash
	vmc.TxRefDataHash = txData

	// RefDataHash (input-specific)
	vmc.RefDataHash = inpData

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
