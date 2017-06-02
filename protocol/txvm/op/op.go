package op

const (
	// control flow & stack
	Fail   = 0
	Verify = 1
	Exec   = 2
	Jump   = 3
	JumpIf = 4
	Roll   = 5 // any stack
	Depth  = 6 // any stack
	Drop   = 7
	Dup    = 8

	// 9

	// data
	List   = 10
	Cons   = 11
	Uncons = 12
	Len    = 13 // list or bytes
	Varint = 14
	Encode = 15

	// boolean
	Equal = 16
	Not   = 17
	And   = 18
	Or    = 19
	GT    = 20
	GE    = 21

	// math
	Abs    = 22
	Add    = 23
	Mul    = 24
	Div    = 25
	Mod    = 26
	Lshift = 27
	Rshift = 28
	Min    = 29
	Max    = 30

	// 31

	// string
	Cat    = 32
	Substr = 33
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

	// 63 - 78

	NumOp = MinInt

	// Small ints.
	// For MinInt <= BaseInt+n < BaseData
	// (so -1 <= n < 16),
	// BaseInt+n pushes n.
	MinInt  = BaseInt - 1
	BaseInt = 80

	BaseData = 96
)
