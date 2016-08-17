package vm

import "errors"

var (
	ErrAltStackUnderflow  = errors.New("alt stack underflow")
	ErrBadValue           = errors.New("bad value")
	ErrCondStackUnderflow = errors.New("conditional stack underflow")
	ErrContext            = errors.New("wrong context")
	ErrDataStackUnderflow = errors.New("data stack underflow")
	ErrDecode             = errors.New("decoding error")
	ErrDivZero            = errors.New("division by zero")
	ErrIllegalOpcode      = errors.New("illegal opcode")
	ErrLoopStackUnderflow = errors.New("loop stack underflow")
	ErrToken              = errors.New("unrecognized token")
	ErrReturn             = errors.New("RETURN executed")
	ErrRunLimitExceeded   = errors.New("run limit exceeded")
	ErrShortProgram       = errors.New("unexpected end of program")
	ErrUnknownHashType    = errors.New("unknown hash type")
	ErrUnknownOpcode      = errors.New("unknown opcode")
	ErrUnsupportedTx      = errors.New("unsupported transaction type")
	ErrUnsupportedVM      = errors.New("unsupported VM")
	ErrVerifyFailed       = errors.New("VERIFY failed")
)
