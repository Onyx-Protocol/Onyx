package vm

// Context contains the execution context for the virtual machine.
//
// Most fields are pointers and are not required to be present in all
// cases. A nil pointer means the value is absent in that context. If
// an opcode executes that requires an absent field to be present, it
// will return ErrContext.
//
// By convention, variables of this type have the name context, _not_
// ctx (to avoid confusion with context.Context).
type Context struct {
	VMVersion uint64
	Code      []byte
	Arguments [][]byte

	// TxVersion must be present when verifying transaction components
	// (such as spends and issuances).
	TxVersion *uint64

	// These fields must be present when verifying block headers.

	BlockHash            *[]byte
	BlockTimeMS          *uint64
	NextConsensusProgram *[]byte

	// Fields below this point are required by particular opcodes when
	// verifying transaction components.

	TxSigHash        *[]byte
	NumResults       *uint64
	AssetID          *[]byte
	Amount           *uint64
	MinTimeMS        *uint64
	MaxTimeMS        *uint64
	InputRefDataHash *[]byte
	TxRefDataHash    *[]byte
	InputIndex       *uint32
	Nonce            *[]byte
	SpentOutputID    *[]byte

	CheckOutput func(index uint64, data []byte, amount uint64, assetID []byte, vmVersion uint64, code []byte) (bool, error)
}
