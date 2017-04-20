package bc

import "io"

// BlockHeader contains the header information for a blockchain
// block. It satisfies the Entry interface.

func (BlockHeader) typ() string { return "blockheader" }
func (bh *BlockHeader) writeForHash(w io.Writer) {
	mustWriteForHash(w, bh.Version)
	mustWriteForHash(w, bh.Height)
	mustWriteForHash(w, bh.PreviousBlockId)
	mustWriteForHash(w, bh.TimestampMs)
	mustWriteForHash(w, bh.TransactionsRoot)
	mustWriteForHash(w, bh.AssetsRoot)
	mustWriteForHash(w, bh.NextConsensusProgram)
	mustWriteForHash(w, bh.ExtHash)
}

// NewBlockHeader creates a new BlockHeader and populates
// its body.
func NewBlockHeader(version, height uint64, previousBlockID *Hash, timestampMS uint64, transactionsRoot, assetsRoot *Hash, nextConsensusProgram []byte) *BlockHeader {
	return &BlockHeader{
		Version:              version,
		Height:               height,
		PreviousBlockId:      previousBlockID,
		TimestampMs:          timestampMS,
		TransactionsRoot:     transactionsRoot,
		AssetsRoot:           assetsRoot,
		NextConsensusProgram: nextConsensusProgram,
	}
}
