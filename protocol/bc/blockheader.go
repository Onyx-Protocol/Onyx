package bc

type blockHeader struct {
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

func (blockHeader) Type() string          { return "blockheader" }
func (bh *blockHeader) Body() interface{} { return bh.body }

func (blockHeader) Ordinal() int { return -1 }

func newBlockHeader(version, height uint64, previousBlockID Hash, timestampMS uint64, transactionsRoot, assetsRoot Hash, nextConsensusProgram []byte) *blockHeader {
	bh := new(blockHeader)
	bh.body.Version = version
	bh.body.Height = height
	bh.body.PreviousBlockID = previousBlockID
	bh.body.TimestampMS = timestampMS
	bh.body.TransactionsRoot = transactionsRoot
	bh.body.AssetsRoot = assetsRoot
	bh.body.NextConsensusProgram = nextConsensusProgram
	return bh
}
