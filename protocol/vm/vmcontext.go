package vm

type VMContext interface {
	VMVersion() uint64
	Code() []byte
	Arguments() [][]byte

	TXVersion() (txVersion uint64, ok bool)

	// BlockHash produces the hash for the current block, and the number
	// of bytes used as input to that hash. It produces an error in a
	// non-block context.
	BlockHash() ([]byte, int64, error)

	// TxSigHash produces the sighash for the current transaction and
	// entry.  It produces an error in a non-transaction context.
	TxSigHash() ([]byte, error)

	NumResults() (uint64, error)

	CheckOutput(index uint32, data []byte, amount uint64, assetID []byte, vmVersion uint64, code []byte) (bool, error)

	AssetID() ([]byte, error)

	Amount() (uint64, error)

	MinTimeMS() (uint64, error)

	MaxTimeMS() (uint64, error)

	EntryData() ([]byte, error)

	TxData() ([]byte, error)

	DestPos() (uint64, error)

	AnchorID() ([]byte, error)

	NextConsensusProgram() ([]byte, error)

	BlockTime() (uint64, error)
}
