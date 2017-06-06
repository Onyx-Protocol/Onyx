package txvm

// These codes identify stacks.
// For example, ROLL reads a stack code
// to select which stack to modify.
const (
	StackData   = 0
	StackAlt    = 1
	StackInput  = 2
	StackValue  = 3
	StackCond   = 4
	StackOutput = 5
	StackNonce  = 6
	StackAnchor = 7
	StackRetire = 8

	StackVM1Value  = 9
	StackVM1Mux    = 10
	StackVM1Nonce  = 11
	StackVM1Anchor = 12
)
