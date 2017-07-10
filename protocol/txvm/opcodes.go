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
	Inspect = 13

	// boolean
	Equal = 14
	Not   = 15
	And   = 16
	Or    = 17

	// math
	Add    = 18
	Sub    = 19
	Mul    = 20
	Div    = 21
	Mod    = 22
	Lshift = 23
	Rshift = 24
	GT     = 25

	// string
	Cat   = 26
	Slice = 27

	// bitwise (int64 or string)
	BitNot = 28
	BitAnd = 29
	BitOr  = 30
	BitXor = 31

	// crypto
	SHA256        = 32
	SHA3          = 33
	CheckSig      = 34
	CheckMultiSig = 35
	PointAdd      = 36 // TODO(kr): review for CA
	PointSub      = 37 // TODO(kr): review for CA
	PointMul      = 38 // TODO(kr): review for CA

	// constructors
	Encode = 39
	Varint = 40

	// Tuple
	MakeTuple = 41
	Untuple   = 42
	Field     = 43

	// introspection
	Type = 44

	// entries
	Annotate     = 45
	Defer        = 46 // prog => cond
	Satisfy      = 47 // cond => {}
	Unlock       = 48 // inputid + data => value + cond
	UnlockOutput = 49 // outputid + data => value + cond
	Merge        = 50 // value value => value
	Split        = 51 // value + amount => value value
	Lock         = 52 // value + prog => outputid
	Retire       = 53 // value + refdata => {}
	Nonce        = 54 // nonce + data => anchor + cond
	Reanchor     = 55
	Issue        = 56 // anchor + data => value + cond
	IssueCA      = 57 // TODO(kr): review for CA
	Before       = 58
	After        = 59
	Summarize    = 60
	Migrate      = 61
	ProveRange   = 62 // TODO(kr): review for CA
	ProveValue   = 63 // TODO(kr): review for CA
	ProveAsset   = 64 // TODO(kr): review for CA
	Blind        = 65 // TODO(kr): review for CA

	// extensions
	Nop0    = 66
	Nop1    = 67
	Nop2    = 68
	Nop3    = 69
	Nop4    = 70
	Nop5    = 71
	Nop6    = 72
	Nop7    = 73
	Nop8    = 74
	Private = 75

	NumOp = 80

	// Small ints.
	// For 0 <= n < 15,
	// opcode BaseInt+n pushes n.
	BaseInt = 80

	BaseData = 95 // data len in [0, 32] has 1-byte len prefix
)
