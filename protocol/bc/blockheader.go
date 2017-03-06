package bc

// BlockHeaderEntry contains the header information for a blockchain
// block. It satisfies the Entry interface.
type BlockHeaderEntry struct {
	Body struct {
		Version              uint64
		Height               uint64
		PreviousBlockID      Hash
		TimestampMS          uint64
		TransactionsRoot     Hash
		AssetsRoot           Hash
		NextConsensusProgram []byte
		ExtHash              Hash
	}
	Witness struct {
		Arguments [][]byte
	}
}

func (BlockHeaderEntry) Type() string          { return "blockheader" }
func (bh *BlockHeaderEntry) body() interface{} { return bh.Body }

func (BlockHeaderEntry) Ordinal() int { return -1 }

// NewBlockHeaderEntry creates a new BlockHeaderEntry and populates
// its body.
func NewBlockHeaderEntry(version, height uint64, previousBlockID Hash, timestampMS uint64, transactionsRoot, assetsRoot Hash, nextConsensusProgram []byte) *BlockHeaderEntry {
	bh := new(BlockHeaderEntry)
	bh.Body.Version = version
	bh.Body.Height = height
	bh.Body.PreviousBlockID = previousBlockID
	bh.Body.TimestampMS = timestampMS
	bh.Body.TransactionsRoot = transactionsRoot
	bh.Body.AssetsRoot = assetsRoot
	bh.Body.NextConsensusProgram = nextConsensusProgram
	return bh
}

// CheckValid does only part of the work of validating a block. The
// rest is handled in ValidateBlock, which calls this.
func (bh *BlockHeaderEntry) CheckValid(_ *validationState) error {
	if bh.Body.Version == 1 && (bh.Body.ExtHash != Hash{}) {
		return errNonemptyExtHash
	}
	return nil
}
