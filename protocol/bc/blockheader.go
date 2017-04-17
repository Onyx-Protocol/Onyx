package bc

// BlockHeader contains the header information for a blockchain
// block. It satisfies the Entry interface.

func (BlockHeader) typ() string           { return "blockheader" }
func (bh *BlockHeader) body() interface{} { return bh.Body }

// NewBlockHeader creates a new BlockHeader and populates
// its body.
func NewBlockHeader(version, height uint64, previousBlockID *Hash, timestampMS uint64, transactionsRoot, assetsRoot *Hash, nextConsensusProgram []byte) *BlockHeader {
	return &BlockHeader{
		Body: &BlockHeader_Body{
			Version:              version,
			Height:               height,
			PreviousBlockId:      previousBlockID,
			TimestampMs:          timestampMS,
			TransactionsRoot:     transactionsRoot,
			AssetsRoot:           assetsRoot,
			NextConsensusProgram: nextConsensusProgram,
		},
		Witness: &BlockHeader_Witness{},
	}
}
