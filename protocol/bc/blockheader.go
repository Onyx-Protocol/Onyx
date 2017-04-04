package bc

// BlockHeaderEntry contains the header information for a blockchain
// block. It satisfies the Entry interface.

func (BlockHeaderEntry) Type() string          { return "blockheader" }
func (bh *BlockHeaderEntry) body() interface{} { return bh.Body }

// NewBlockHeaderEntry creates a new BlockHeaderEntry and populates
// its body.
func NewBlockHeaderEntry(version, height uint64, previousBlockID Hash, timestampMS uint64, transactionsRoot, assetsRoot Hash, nextConsensusProgram []byte) *BlockHeaderEntry {
	bh := new(BlockHeaderEntry)
	bh.Body.Version = version
	bh.Body.Height = height
	bh.Body.PreviousBlockId = previousBlockID.Proto()
	bh.Body.TimestampMs = timestampMS
	bh.Body.TransactionsRoot = transactionsRoot.Proto()
	bh.Body.AssetsRoot = assetsRoot.Proto()
	bh.Body.NextConsensusProgram = nextConsensusProgram
	return bh
}
