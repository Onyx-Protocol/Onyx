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

// TxHashesFunc is initialized to the function of the same name
// in chain/protocol/tx.
// It is a variable here to avoid a circular dependency
// between package bc and package tx.
// TODO: find a better name for this
// (obvious name is TxHashes, same as the type)
var TxHashesFunc func(*TxData) (*TxHashes, error)

func (t TxHashes) SigHash(n uint32) Hash {
	return t.VMContexts[n].TxSigHash
}

// BlockHeaderHashFunc is initialized to a function in protocol/tx
// that can compute the hash of a blockheader. It is a variable here
// to avoid a circular dependency between the bc and tx packages.
var BlockHeaderHashFunc func(*BlockHeader) Hash

// OutputHash is initialized to a function in protocol/tx
// that can compute the hash of an output from a SpendCommitment.
// It is a variable here to avoid a circular dependency between
// the bc and tx packages.
var OutputHash func(*SpendCommitment) (Hash, error)
