package op

const (
	// control flow
	Fail   = 0
	PC     = 1
	Exec   = 2
	Jump   = 3
	JumpIf = 4

	// stack
	Roll    = 5 // any stack
	Bury    = 6 // any stack
	Depth   = 7 // any stack
	Drop    = 8
	Dup     = 9
	ToAlt   = 10
	FromAlt = 11

	// data
	List   = 12
	Cons   = 13
	Uncons = 14
	Len    = 15 // list or string
	Varint = 16
	Encode = 17

	// boolean
	Equal = 18
	Not   = 19
	And   = 20
	Or    = 21
	GT    = 22
	GE    = 23

	// math
	Abs    = 24
	Add    = 25
	Mul    = 26
	Div    = 27
	Mod    = 28
	Lshift = 29
	Rshift = 30
	Min    = 31
	Max    = 32

	// string
	Cat   = 33
	Slice = 34

	// bitwise (int64 or string)
	BitNot = 35
	BitAnd = 36
	BitOr  = 37
	BitXor = 38

	// crypto
	SHA256        = 39
	SHA3          = 40
	CheckSig      = 41
	CheckMultiSig = 42
	PointAdd      = 43 // TODO(kr): review for CA
	PointSub      = 44 // TODO(kr): review for CA
	PointMul      = 45 // TODO(kr): review for CA

	// entries
	Cond         = 46 // prog => cond
	Unlock       = 47 // inputid + data => value + cond
	UnlockOutput = 48 // outputid + data => value + cond
	Merge        = 49 // value value => value
	Split        = 50 // value + amount => value value
	ProveRange   = 51 // TODO(kr): review for CA
	ProveValue   = 52 // TODO(kr): review for CA
	ProveAsset   = 53 // TODO(kr): review for CA
	Blind        = 54 // TODO(kr): review for CA
	Lock         = 55 // value + prog => outputid
	Satisfy      = 56 // cond => {}
	Anchor       = 57 // nonce + data => anchor + cond
	Issue        = 58 // anchor + data => value + cond
	IssueCA      = 59 // TODO(kr): review for CA
	Retire       = 60 // valud + refdata => {}

	// item data
	ID    = 61
	Value = 62 // TODO(kr): name for this? "AssetAmount"?

	// compatbility
	VM1Nonce          = 63 // vm1nonce => vm1anchor + cond
	VM1Issue          = 64 // vm1anchor => vm1value + cond
	VM1Unlock         = 65 // vm1inputid + data => vm1value + cond
	VM1Mux            = 66 // entire vm1value stack => vm1mux
	VM1Withdraw       = 67 // vm1mux + amount asset => vm1mux + value
	VM1CheckPredicate = 68 // list vm1prog => bool

	// extensions
	Nop0    = 69
	Nop1    = 70
	Nop2    = 71
	Nop3    = 72
	Nop4    = 73
	Nop5    = 74
	Nop6    = 75
	Nop7    = 76
	Nop8    = 77
	Private = 78

	NumOp = 79

	// Small ints.
	// For MinInt <= BaseInt+n < BaseData
	// (so -1 <= n < 15),
	// opcode BaseInt+n pushes n.
	MinInt  = 79
	BaseInt = 80

	BaseData = 95 // data len in [0, 32] has 1-byte len prefix
)
