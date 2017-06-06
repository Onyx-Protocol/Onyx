package txvm

const (
	// control flow
	Fail   = 0
	PC     = 1
	Exec   = 2
	JumpIf = 3

	// stack
	Roll  = 4 // any stack
	Bury  = 5 // any stack
	Depth = 6 // any stack

	// Data Stack
	Equal   = 7
	Type    = 8
	Encode  = 9
	Len     = 10
	Drop    = 11
	Dup     = 12
	ToAlt   = 13
	FromAlt = 14

	// Tuple
	Tuple   = 15
	Untuple = 16
	Field   = 17

	// boolean
	Not = 18
	And = 19
	Or  = 20
	GT  = 21
	GE  = 22

	// math
	Add    = 23
	Mul    = 24
	Div    = 25
	Mod    = 26
	Lshift = 27
	Rshift = 28
	Negate = 29

	// string
	Cat   = 30
	Slice = 31

	// bitwise (int64 or string)
	BitNot = 32
	BitAnd = 33
	BitOr  = 34
	BitXor = 35

	// crypto
	SHA256        = 36
	SHA3          = 37
	CheckSig      = 38
	CheckMultiSig = 39
	PointAdd      = 40 // TODO(kr): review for CA
	PointSub      = 41 // TODO(kr): review for CA
	PointMul      = 42 // TODO(kr): review for CA

	// entries
	Cond         = 43 // prog => cond
	Unlock       = 44 // inputid + data => value + cond
	UnlockOutput = 45 // outputid + data => value + cond
	Merge        = 46 // value value => value
	Split        = 47 // value + amount => value value
	ProveRange   = 48 // TODO(kr): review for CA
	ProveValue   = 49 // TODO(kr): review for CA
	ProveAsset   = 50 // TODO(kr): review for CA
	Blind        = 51 // TODO(kr): review for CA
	Lock         = 52 // value + prog => outputid
	Satisfy      = 53 // cond => {}
	Anchor       = 54 // nonce + data => anchor + cond
	Issue        = 55 // anchor + data => value + cond
	IssueCA      = 56 // TODO(kr): review for CA
	Retire       = 57 // value + refdata => {}

	// compatibility
	VM1CheckPredicate = 58 // list vm1prog => bool
	VM1Unlock         = 59 // vm1inputid + data => vm1value + cond
	VM1Nonce          = 60 // vm1nonce => vm1anchor + cond
	VM1Issue          = 61 // vm1anchor => vm1value + cond
	VM1Mux            = 62 // entire vm1value stack => vm1mux
	VM1Withdraw       = 63 // vm1mux + amount asset => vm1mux + value

	// 64-68

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

	// constructors
	Varint = 79

	NumOp = 80

	// Small ints.
	// For MinInt <= BaseInt+n < BaseData
	// (so 0 <= n <= 15),
	// opcode BaseInt+n pushes n.
	MinInt  = 80
	BaseInt = 80

	BaseData = 96 // data len in [0, 32] has 1-byte len prefix
)
