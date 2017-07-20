package txvm2

import (
	"encoding/binary"
	"fmt"
	"math"

	"chain/math/checked"
)

//go:generate go run gen.go

// This file is read by gen.go at "go generate" time and produces
// opgen.go.

// TODO: when the spec is final, freeze the numeric values of these constants

const (
	// Control flow
	OpFail byte = 33 + iota // opcodes 0 through 32 reserved for small ints
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
	OpWrapValue
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

	// Encoding
	OpEncode
	OpInt64
	OpPushdata // xxx this is not the spec, will change

	// Conversion
	OpFinalize
	OpUnlockLegacy
	OpIssueLegacy
	OpLegacyIssuanceCandidate
	OpExtend
)

const (
	Op0         byte = 0
	MaxSmallInt byte = 32
)

func init() {
	if OpFail != MaxSmallInt+1 {
		panic(fmt.Errorf("OpFail is %d, should be %d", OpFail, MaxSmallInt+1))
	}
	opFuncs[OpSatisfy] = opSatisfy
	opFuncs[OpCommand] = opCommand
	opFuncs[OpProveAssetRange] = opProveAssetRange
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
	if uint64(len(prog)) < l {
		return nil, 0, fmt.Errorf("pushdata: only %d of %d bytes available", len(prog), l)
	}
	consumed, ok := checked.AddInt64(int64(n), int64(l))
	if !ok {
		return nil, 0, fmt.Errorf("pushdata: bytes consumed overflows int64")
	}
	return prog[:l], consumed, nil
}

func encodePushdata(data []byte) []byte {
	buf := [12]byte{OpPushdata}
	n := binary.PutUvarint(buf[1:], uint64(len(data)))
	return buf[:1+n]
}
