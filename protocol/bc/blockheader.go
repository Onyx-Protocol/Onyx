package bc

// BlockHeaderEntry contains the header information for a blockchain
// block. It satisfies the Entry interface.

func (BlockHeaderEntry) Type() string          { return "blockheader" }
func (bh *BlockHeaderEntry) body() interface{} { return bh.Body }

// NewBlockHeaderEntry creates a new BlockHeaderEntry and populates
// its body.
func NewBlockHeaderEntry(version, height uint64, previousBlockID Hash, timestampMS uint64, transactionsRoot, assetsRoot Hash, nextConsensusProgram []byte) *BlockHeaderEntry {
	return &BlockHeaderEntry{
		Body: &BlockHeaderEntry_Body{
			Version:              version,
			Height:               height,
			PreviousBlockId:      previousBlockID.Proto(),
			TimestampMs:          timestampMS,
			TransactionsRoot:     transactionsRoot.Proto(),
			AssetsRoot:           assetsRoot.Proto(),
			NextConsensusProgram: nextConsensusProgram,
		},
		Witness: &BlockHeaderEntry_Witness{},
	}
}
