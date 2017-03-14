package bc

type validationState struct {
	blockVersion      uint64
	txVersion         uint64
	initialBlockID    Hash
	currentEntryID    Hash
	sourcePosition    uint64
	destPosition      uint64
	timestampMS       uint64
	prevBlockHeader   *BlockHeaderEntry
	prevBlockHeaderID Hash
	blockTxs          []*TxEntries
	// xxx reachable entries?
}
