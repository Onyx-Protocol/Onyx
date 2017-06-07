package txvm

//go:generate sh gennumber.sh
// Generation is temporary, should be removed once package stabilizes

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
	Len     = 7
	Drop    = 8
	Dup     = 9
	ToAlt   = 10
	FromAlt = 11

	// boolean
	Equal = 12
	Not   = 13
	And   = 14
	Or    = 15
	GT    = 16
	GE    = 17

	// math
	Add    = 18
	Mul    = 19
	Div    = 20
	Mod    = 21
	Lshift = 22
	Rshift = 23
	Negate = 24

	// string
	Cat   = 25
	Slice = 26

	// bitwise (int64 or string)
	BitNot = 27
	BitAnd = 28
	BitOr  = 29
	BitXor = 30

	// crypto
	SHA256        = 31
	SHA3          = 32
	CheckSig      = 33
	CheckMultiSig = 34
	PointAdd      = 35 // TODO(kr): review for CA
	PointSub      = 36 // TODO(kr): review for CA
	PointMul      = 37 // TODO(kr): review for CA

	// constructors
	Encode = 38
	Varint = 39

	// Tuple
	Tuple   = 40
	Untuple = 41
	Field   = 42

	// introspection
	Type = 43

	// entries
	Cond         = 44 // prog => cond
	Unlock       = 45 // inputid + data => value + cond
	UnlockOutput = 46 // outputid + data => value + cond
	Merge        = 47 // value value => value
	Split        = 48 // value + amount => value value
	ProveRange   = 49 // TODO(kr): review for CA
	ProveValue   = 50 // TODO(kr): review for CA
	ProveAsset   = 51 // TODO(kr): review for CA
	Blind        = 52 // TODO(kr): review for CA
	Lock         = 53 // value + prog => outputid
	Satisfy      = 54 // cond => {}
	Anchor       = 55 // nonce + data => anchor + cond
	Issue        = 56 // anchor + data => value + cond
	IssueCA      = 57 // TODO(kr): review for CA
	Retire       = 58 // value + refdata => {}

	// compatibility
	VM1CheckPredicate = 59 // list vm1prog => bool
	VM1Unlock         = 60 // vm1inputid + data => vm1value + cond
	VM1Nonce          = 61 // vm1nonce => vm1anchor + cond
	VM1Issue          = 62 // vm1anchor => vm1value + cond
	VM1Mux            = 63 // entire vm1value stack => vm1mux
	VM1Withdraw       = 64 // vm1mux + amount asset => vm1mux + value

	// extensions
	Nop0    = 65
	Nop1    = 66
	Nop2    = 67
	Nop3    = 68
	Nop4    = 69
	Nop5    = 70
	Nop6    = 71
	Nop7    = 72
	Nop8    = 73
	Private = 74

	NumOp = 80

	// Small ints.
	// For MinInt <= BaseInt+n < BaseData
	// (so 0 <= n < 15),
	// opcode BaseInt+n pushes n.
	MinInt  = 80
	BaseInt = 80

	BaseData = 95 // data len in [0, 32] has 1-byte len prefix
)
