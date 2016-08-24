package vm

import "errors"

var (
	ErrAltStackUnderflow     = errors.New("alt stack underflow")
	ErrBadControlSyntax      = errors.New("control flow syntax error")
	ErrBadValue              = errors.New("bad value")
	ErrContext               = errors.New("wrong context")
	ErrControlStackUnderflow = errors.New("control flow stack underflow")
	ErrDataStackUnderflow    = errors.New("data stack underflow")
	ErrDecode                = errors.New("decoding error")
	ErrDivZero               = errors.New("division by zero")
	ErrNonEmptyControlStack  = errors.New("unterminated control flow")
	ErrRange                 = errors.New("range error")
	ErrReturn                = errors.New("RETURN executed")
	ErrRunLimitExceeded      = errors.New("run limit exceeded")
	ErrShortProgram          = errors.New("unexpected end of program")
	ErrToken                 = errors.New("unrecognized token")
	ErrUnknownHashType       = errors.New("unknown hash type")
	ErrUnknownOpcode         = errors.New("unknown opcode")
	ErrUnsupportedTx         = errors.New("unsupported transaction type")
	ErrUnsupportedVM         = errors.New("unsupported VM")
	ErrVerifyFailed          = errors.New("VERIFY failed")
)
