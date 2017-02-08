package errors

import (
	"fmt"
	"runtime"
)

const stackTraceSize = 10

// StackFrame represents a single entry in a stack trace.
type StackFrame struct {
	Func string
	File string
	Line int
}

// String satisfies the fmt.Stringer interface.
func (f StackFrame) String() string {
	return fmt.Sprintf("%s:%d - %s", f.File, f.Line, f.Func)
}

// Stack returns the stack trace of an error. The error must contain the stack
// trace, or wrap an error that has a stack trace,
func Stack(err error) []StackFrame {
	if wErr, ok := err.(wrapperError); ok {
		return wErr.stack
	}
	return nil
}

// getStack is a formatting wrapper around runtime.Callers. It returns a stack
// trace in the form of a StackFrame slice.
func getStack(skip int, size int) []StackFrame {
	var (
		pc    = make([]uintptr, size)
		calls = runtime.Callers(skip+1, pc)
		trace []StackFrame
	)

	for i := 0; i < calls; i++ {
		f := runtime.FuncForPC(pc[i])
		file, line := f.FileLine(pc[i] - 1)
		trace = append(trace, StackFrame{
			Func: f.Name(),
			File: file,
			Line: line,
		})
	}

	return trace
}
