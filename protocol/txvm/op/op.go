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

	// 46

	// entries
	Cond         = 47 // prog => cond
	Unlock       = 48 // inputid + data => value + cond
	UnlockOutput = 49 // outputid + data => value + cond
	Merge        = 50 // value value => value
	Split        = 51 // value + amount => value value
	Lock         = 52 // value + prog => outputid
	Satisfy      = 53 // cond => {}
	Anchor       = 54 // nonce + data => anchor + cond
	Issue        = 55 // anchor + data => value + cond
	Retire       = 56 // valud + refdata => {}

	// compatbility
	VM1Nonce          = 57 // vm1nonce => vm1anchor + cond
	VM1Issue          = 58 // vm1anchor => vm1value + cond
	VM1Unlock         = 60 // vm1inputid + data => vm1value + cond
	VM1Mux            = 59 // entire vm1value stack => vm1mux
	VM1Withdraw       = 61 // vm1mux + amount asset => vm1mux + value
	VM1CheckPredicate = 62 // list vm1prog => bool

	// extensions
	Private = 63
	Nop0    = 64
	Nop1    = 65
	Nop2    = 66
	Nop3    = 67

	// CA-specific entries
	ProveRange = 68 // TODO(kr): review for CA
	ProveValue = 69 // TODO(kr): review for CA
	ProveAsset = 70 // TODO(kr): review for CA
	Blind      = 71 // TODO(kr): review for CA
	IssueCA    = 72 // TODO(kr): review for CA

	// 73 - 78

	NumOp = MinInt

	// Small ints.
	// For MinInt <= BaseInt+n < BaseData
	// (so -1 <= n < 15),
	// BaseInt+n pushes n.
	MinInt  = BaseInt - 1
	BaseInt = 80

	BaseData = 95 // data len in [0, 32] has 1-byte len prefix
)
