package bc

type (
	// TxHashes holds data needed for validation and state updates.
	TxHashes struct {
		ID Hash

		// contains OutputIDs and retirement hashes.
		Results   []ResultInfo
		Issuances []struct {
			ID           Hash
			ExpirationMS uint64
		}
		SpentOutputIDs []Hash       // one per old-style Input. Non-spend inputs are blank hashes.
		VMContexts     []*VMContext // one per old-style Input
	}

	// ResultInfo contains information about each result in a transaction header.
	ResultInfo struct {
		ID Hash // outputID

		// The following fields apply only to results that are outputs (not retirements).
		SourceID    Hash   // the ID of this output's source entry
		SourcePos   uint64 // the position within the source entry of this output's value
		RefDataHash Hash   // contents of the result entry's data field (which is a hash of the source refdata, when converting from old-style transactions)
	}

	VMContext struct {
		TxRefDataHash Hash
		RefDataHash   Hash
		TxSigHash     Hash
		OutputID      *Hash
		EntryID       Hash
		NonceID       *Hash
	}
)

func (t TxHashes) SigHash(n uint32) Hash {
	return t.VMContexts[n].TxSigHash
}
