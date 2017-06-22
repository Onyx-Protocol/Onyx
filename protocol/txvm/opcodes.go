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
	Sub    = 17
	Mul    = 18
	Div    = 19
	Mod    = 20
	Lshift = 21
	Rshift = 22
	GT     = 23

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
	MakeTuple = 39
	Untuple   = 40
	Field     = 41

	// introspection
	Type = 42

	// entries
	Annotate     = 43
	Defer        = 44 // prog => cond
	Satisfy      = 45 // cond => {}
	Unlock       = 46 // inputid + data => value + cond
	UnlockOutput = 47 // outputid + data => value + cond
	Merge        = 48 // value value => value
	Split        = 49 // value + amount => value value
	Lock         = 50 // value + prog => outputid
	Retire       = 51 // value + refdata => {}
	Anchor       = 52 // nonce + data => anchor + cond
	Issue        = 53 // anchor + data => value + cond
	IssueCA      = 54 // TODO(kr): review for CA
	Header       = 55
	ProveRange   = 56 // TODO(kr): review for CA
	ProveValue   = 57 // TODO(kr): review for CA
	ProveAsset   = 58 // TODO(kr): review for CA
	Blind        = 59 // TODO(kr): review for CA

	// extensions
	Nop0    = 60
	Nop1    = 61
	Nop2    = 62
	Nop3    = 63
	Nop4    = 64
	Nop5    = 65
	Nop6    = 66
	Nop7    = 67
	Nop8    = 68
	Private = 69

	NumOp = 80

	// Small ints.
	// For 0 <= n < 15,
	// opcode BaseInt+n pushes n.
	BaseInt = 80

	BaseData = 95 // data len in [0, 32] has 1-byte len prefix
)
