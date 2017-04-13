package bc

type Block struct {
	*BlockHeader
	ID           Hash
	Transactions []*Tx
}
