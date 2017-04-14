package bc

type BlockEntries struct {
	*BlockHeaderEntry
	ID           Hash
	Transactions []*TxEntries
}
