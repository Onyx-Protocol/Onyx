package txvm

//go:generate sh gennumber.sh
// Generation is temporary, should be removed once package stabilizes

const (
	// control flow
	Fail   = 0
	PC     = 1
	JumpIf = 2

	// stack
	Roll    = 3 // any stack
	Bury    = 4 // any stack
	Reverse = 5
	Depth   = 6 // any stack
	ID      = 7

	// Data Stack
	Len     = 8
	Drop    = 9
	Dup     = 10
	ToAlt   = 11
	FromAlt = 12

	// boolean
	Equal = 13
	Not   = 14
	And   = 15
	Or    = 16

	// math
	Add    = 17
	Sub    = 18
	Mul    = 19
	Div    = 20
	Mod    = 21
	Lshift = 22
	Rshift = 23
	GT     = 24

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
	MakeTuple = 40
	Untuple   = 41
	Field     = 42

	// introspection
	Type = 43

	// entries
	Annotate     = 44
	Defer        = 45 // prog => cond
	Satisfy      = 46 // cond => {}
	Unlock       = 47 // inputid + data => value + cond
	UnlockOutput = 48 // outputid + data => value + cond
	Merge        = 49 // value value => value
	Split        = 50 // value + amount => value value
	Lock         = 51 // value + prog => outputid
	Retire       = 52 // value + refdata => {}
	Anchor       = 53 // nonce + data => anchor + cond
	Issue        = 54 // anchor + data => value + cond
	IssueCA      = 55 // TODO(kr): review for CA
	Before       = 56
	After        = 57
	Summarize    = 58
	Migrate      = 59
	ProveRange   = 60 // TODO(kr): review for CA
	ProveValue   = 61 // TODO(kr): review for CA
	ProveAsset   = 62 // TODO(kr): review for CA
	Blind        = 63 // TODO(kr): review for CA

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
