package bc

type (
	// TxHashes holds data needed for validation and state updates.
	TxHashes struct {
		ID        Hash
		OutputIDs []Hash // each OutputID is also the corresponding UnspentID
		Issuances []struct {
			ID           Hash
			ExpirationMS uint64
		}
		VMContexts []*VMContext // one per old-style Input
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
var BlockHeaderHashFunc func(*BlockHeader) (Hash, error)
