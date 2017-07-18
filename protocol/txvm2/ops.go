package txvm2

//go:generate go run gen.go

// This file is used as input to gen.go, which is used by "go
// generate" to produce other files in this package. The first
// declaration in this file must be the list of opcode constants.

// TODO: when the spec is final, freeze the numeric values of these constants

const (
	// Control flow
	OpFail byte = iota
	OpPC
	OpJumpIf

	// Stacks
	OpRoll
	OpBury
	OpReverse
	OpDepth
	OpPeek

	// Data stack
	OpEqual
	OpType
	OpLen
	OpDrop
	OpToAlt
	OpFromAlt

	// Tuples
	OpTuple
	OpUntuple
	OpField

	// Booleans
	OpNot
	OpAnd
	OpOr

	// Math
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpLeftShift
	OpRightShift
	OpGreaterThan

	// Strings
	OpCat
	OpSlice

	// Bitwise
	OpBitNot
	OpBitAnd
	OpBitOr
	OpBitXor

	// Crypto
	OpSHA256
	OpSHA3
	OpCheckSig
	OpPointAdd
	OpPointSub
	OpPointMul

	// Annotation
	OpAnnotate

	// Commands
	OpCommand

	// Conditions
	OpDefer
	OpSatisfy

	// Records
	OpCreate
	OpDelete
	OpComplete

	// "Contracts"
	// TODO: need a different name
	OpUnlock
	OpRead
	OpLock

	// Values
	OpIssue
	OpMerge
	OpSplit
	OpRetire

	// Confidentiality
	OpMergeConfidential
	OpSplitConfidential
	OpProveAssetRange
	OpDropAssetCommitment
	OpProveAssetID
	OpProveAmount
	OpProveValueRange
	OpIssuanceCandidate
	OpIssueConfidential

	// Anchors
	OpNonce
	OpReanchor
	OpSplitAnchor
	OpAnchorTransaction

	// Times
	OpBefore
	OpAfter

	// Conversion
	OpFinalize
	OpUnlockLegacy
	OpIssueLegacy
	OpLegacyIssuanceCandidate
	OpExtend

	// Encoding
	OpEncode
	OpInt64
	OpPushdata
	Op0
	MaxSmallInt = Op0 + 32
	NumOps      = MaxSmallInt + 1
)

func init() {
	opFuncs[OpSatisfy] = opSatisfy
	opFuncs[OpCommand] = opCommand
}

func isSmallIntOp(op byte) bool {
	return op >= Op0 && op <= MaxSmallInt
}

func decodeInst(prog []byte) (opcode byte, data []byte, delta int64) {
	opcode = prog[0]
	delta = 1
	switch opcode {
	// xxx
	}
	return // xxx
}
