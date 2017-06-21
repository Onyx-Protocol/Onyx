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
	ID    = 6

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

	// math
	Add    = 16
	Mul    = 17
	Div    = 18
	Mod    = 19
	Lshift = 20
	Rshift = 21
	Negate = 22
	GT     = 23
	GE     = 24

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

	// extensions
	Nop0    = 59
	Nop1    = 60
	Nop2    = 61
	Nop3    = 62
	Nop4    = 63
	Nop5    = 64
	Nop6    = 65
	Nop7    = 66
	Nop8    = 67
	Private = 68

	NumOp = 80

	// Small ints.
	// For 0 <= n < 15,
	// opcode BaseInt+n pushes n.
	BaseInt = 80

	BaseData = 95 // data len in [0, 32] has 1-byte len prefix
)
