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
	o := NewOutput(Program{VMVersion: sc.VMVersion, Code: sc.ControlProgram}, sc.RefDataHash, 0)
	o.setSourceID(sc.SourceID, sc.AssetAmount, sc.SourcePosition)

	h = EntryID(o)
	return h, nil
}

// ComputeTxHashes returns all hashes needed for validation and state updates.
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
	hashes.Results = make([]ResultInfo, len(header.Body.Results))
	for i, resultHash := range header.Body.Results {
		hashes.Results[i].ID = resultHash
		entry := entries[resultHash]
		if out, ok := entry.(*Output); ok {
			hashes.Results[i].SourceID = out.Body.Source.Ref
			hashes.Results[i].SourcePos = out.Body.Source.Position
			hashes.Results[i].RefDataHash = out.Body.Data
		}
	}

	hashes.SpentOutputIDs = make([]Hash, len(oldTx.Inputs))
	hashes.SigHashes = make([]Hash, len(oldTx.Inputs))

	for entryID, ent := range entries {
		switch ent := ent.(type) {
		case *Nonce:
			// TODO: check time range is within network-defined limits
			trID := ent.Body.TimeRange
			trEntry, ok := entries[trID]
			if !ok {
				return nil, fmt.Errorf("nonce entry refers to nonexistent timerange entry")
			}
			tr, ok := trEntry.(*TimeRange)
			if !ok {
				return nil, fmt.Errorf("nonce entry refers to %s entry, should be timerange", trEntry.Type())
			}
			iss := struct {
				ID           Hash
				ExpirationMS uint64
			}{entryID, tr.Body.MaxTimeMS}
			hashes.Issuances = append(hashes.Issuances, iss)

		case *Issuance:
			hashes.SigHashes[ent.Ordinal()] = makeSigHash(entryID, hashes.ID)

		case *Spend:
			hashes.SigHashes[ent.Ordinal()] = makeSigHash(entryID, hashes.ID)
			hashes.SpentOutputIDs[ent.Ordinal()] = ent.Body.SpentOutput
		}
	}

	return hashes, nil
}

func makeSigHash(entryID, txID Hash) (hash Hash) {
	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)
	hasher.Write(entryID[:])
	hasher.Write(txID[:])
	hasher.Read(hash[:])
	return hash
}
