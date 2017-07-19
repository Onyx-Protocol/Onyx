package txvm2

import (
	"encoding/binary"
	"fmt"
	"math"
)

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
	OpPushdata // xxx this is not the spec, will change
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

// prog is the slice beginning right after a pushdata instruction.
// returns the data parsed and the number of bytes consumed (counting
// the length prefix and the data).
func decodePushdata(prog []byte) ([]byte, int64, error) {
	l, n := binary.Uvarint(prog)
	if n == 0 {
		return nil, 0, fmt.Errorf("pushdata: unexpected end of input reading length prefix")
	}
	if n < 0 {
		return nil, 0, fmt.Errorf("pushdata: length overflow")
	}
	if l > math.MaxInt64 {
		return nil, 0, fmt.Errorf("pushdata: length %d exceeds maximum of %d", l, math.MaxInt64)
	}
	prog = prog[n:]
	if len(prog) < l {
		return nil, 0, fmt.Errorf("pushdata: only %d of %d bytes available", len(prog), l)
	}
	return prog[:l], n + l, err
}
