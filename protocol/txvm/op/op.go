package op

const (
	// control flow
	Fail   = 0
	Verify = 1
	PC     = 2
	Exec   = 3
	Jump   = 4
	JumpIf = 5

	// stack
	Roll    = 6 // any stack
	Bury    = 7 // any stack
	Depth   = 8 // any stack
	Drop    = 9
	Dup     = 10
	ToAlt   = 11
	FromAlt = 12

	// data
	List   = 13
	Cons   = 14
	Uncons = 15
	Len    = 16 // list or string
	Varint = 17
	Encode = 18

	// boolean
	Equal = 19
	Not   = 20
	And   = 21
	Or    = 22
	GT    = 23
	GE    = 24

	// math
	Abs    = 25
	Add    = 26
	Mul    = 27
	Div    = 28
	Mod    = 29
	Lshift = 30
	Rshift = 31
	Min    = 32
	Max    = 33

	// string
	Cat   = 34
	Slice = 35

	// bitwise (int64 or string)
	BitNot = 36
	BitAnd = 37
	BitOr  = 38
	BitXor = 39

	// crypto
	SHA256        = 40
	SHA3          = 41
	CheckSig      = 42
	CheckMultiSig = 43

	// 44 - 46

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

	// 67 - 78

	NumOp = MinInt

	// Small ints.
	// For MinInt <= BaseInt+n < BaseData
	// (so -1 <= n < 16),
	// BaseInt+n pushes n.
	MinInt  = BaseInt - 1
	BaseInt = 80

	BaseData = 96
)
