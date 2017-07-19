package errors

import "runtime"

const stackTraceSize = 10

// Stack returns the stack trace of an error.
// If the error has no stack information, it
// returns an empty stack trace.
func Stack(err error) *runtime.Frames {
	wErr, _ := err.(wrapperError)
	return runtime.CallersFrames(wErr.stack)
}

// getStack is an allocating wrapper around runtime.Callers.
func getStack(skip int, max int) []uintptr {
	pcs := make([]uintptr, max)
	return pcs[:runtime.Callers(skip+1, pcs)]
}
