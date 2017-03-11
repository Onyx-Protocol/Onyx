package bc

// BlockHeaderEntry contains the header information for a blockchain
// block. It satisfies the Entry interface.
type BlockHeaderEntry struct {
	body struct {
		Version              uint64
		Height               uint64
		PreviousBlockID      Hash
		TimestampMS          uint64
		TransactionsRoot     Hash
		AssetsRoot           Hash
		NextConsensusProgram []byte
		ExtHash              Hash
	}
}

func (BlockHeaderEntry) Type() string          { return "blockheader" }
func (bh *BlockHeaderEntry) Body() interface{} { return bh.body }

func (BlockHeaderEntry) Ordinal() int { return -1 }

// NewBlockHeaderEntry creates a new BlockHeaderEntry and populates
// its body.
func NewBlockHeaderEntry(version, height uint64, previousBlockID Hash, timestampMS uint64, transactionsRoot, assetsRoot Hash, nextConsensusProgram []byte) *BlockHeaderEntry {
	bh := new(BlockHeaderEntry)
	bh.body.Version = version
	bh.body.Height = height
	bh.body.PreviousBlockID = previousBlockID
	bh.body.TimestampMS = timestampMS
	bh.body.TransactionsRoot = transactionsRoot
	bh.body.AssetsRoot = assetsRoot
	bh.body.NextConsensusProgram = nextConsensusProgram
	return bh
}
