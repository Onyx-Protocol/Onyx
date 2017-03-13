package log

import (
	"path/filepath"
	"runtime"
	"strconv"
)

var skipFunc = map[string]bool{
	"chain/log.Printkv":            true,
	"chain/log.Printf":             true,
	"chain/log.Error":              true,
	"chain/log.Fatalkv":            true,
	"chain/log.RecoverAndLogError": true,
}

// SkipFunc removes the named function from stack traces
// and at=[file:line] entries printed to the log output.
// The provided name should be a fully-qualified function name
// comprising the import path and identifier separated by a dot.
// For example, chain/log.Printkv.
// SkipFunc must not be called concurrently with any function
// in this package (including itself).
func SkipFunc(name string) {
	skipFunc[name] = true
}

// caller returns a string containing filename and line number of
// the deepest function invocation on the calling goroutine's stack,
// after skipping functions in skipFunc.
// If no stack information is available, it returns "?:?".
func caller() string {
	for i := 1; ; i++ {
		// NOTE(kr): This is quadratic in the number of frames we
		// ultimately have to skip. Consider using Callers instead.
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			return "?:?"
		}
		if !skipFunc[runtime.FuncForPC(pc).Name()] {
			return filepath.Base(file) + ":" + strconv.Itoa(line)
		}
	}
}
