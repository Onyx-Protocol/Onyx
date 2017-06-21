package txvm

//go:generate sh gennumber.sh
// Generation is temporary, should be removed once package stabilizes

const (
	// control flow
	Fail   = 0
	PC     = 1
	JumpIf = 2

	// stack
	Roll  = 3 // any stack
	Bury  = 4 // any stack
	Depth = 5 // any stack

	// Data Stack
	Len     = 6
	Drop    = 7
	Dup     = 8
	ToAlt   = 9
	FromAlt = 10

	// boolean
	Equal = 11
	Not   = 12
	And   = 13
	Or    = 14

	// math
	Add    = 15
	Mul    = 16
	Div    = 17
	Mod    = 18
	Lshift = 19
	Rshift = 20
	Negate = 21
	GT     = 22
	GE     = 23

	// string
	Cat   = 24
	Slice = 25

	// bitwise (int64 or string)
	BitNot = 26
	BitAnd = 27
	BitOr  = 28
	BitXor = 29

	// crypto
	SHA256        = 30
	SHA3          = 31
	CheckSig      = 32
	CheckMultiSig = 33
	PointAdd      = 34 // TODO(kr): review for CA
	PointSub      = 35 // TODO(kr): review for CA
	PointMul      = 36 // TODO(kr): review for CA

	// constructors
	Encode = 37
	Varint = 38

	// Tuple
	Tuple   = 39
	Untuple = 40
	Field   = 41

	// introspection
	Type = 42

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

	// extensions
	Nop0    = 64
	Nop1    = 65
	Nop2    = 66
	Nop3    = 67
	Nop4    = 68
	Nop5    = 69
	Nop6    = 70
	Nop7    = 71
	Nop8    = 72
	Private = 73

	NumOp = 80

	// Small ints.
	// For 0 <= n < 15,
	// opcode BaseInt+n pushes n.
	BaseInt = 80

	BaseData = 95 // data len in [0, 32] has 1-byte len prefix
)
