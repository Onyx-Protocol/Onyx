package op

const (
	// control flow & stack
	Fail   = 0
	Verify = 1
	PC     = 2
	Exec   = 3
	Jump   = 4
	JumpIf = 5
	Roll   = 6 // any stack
	Bury   = 7 // any stack
	Depth  = 8 // any stack
	Drop   = 9
	Dup    = 10

	// data
	List   = 11
	Cons   = 12
	Uncons = 13
	Len    = 14 // list or string
	Varint = 15
	Encode = 16

	// boolean
	Equal = 17
	Not   = 18
	And   = 19
	Or    = 20
	GT    = 21
	GE    = 22

	// math
	Abs    = 23
	Add    = 24
	Mul    = 25
	Div    = 26
	Mod    = 27
	Lshift = 28
	Rshift = 29
	Min    = 30
	Max    = 31

	// string
	Cat   = 32
	Slice = 33

	// bitwise (int64 or string)
	BitNot = 34
	BitAnd = 35
	BitOr  = 36
	BitXor = 37

	// 38 - 41

	// crypto
	SHA256        = 42
	SHA3          = 43
	CheckSig      = 44
	CheckMultiSig = 45

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
