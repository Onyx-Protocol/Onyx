package span

import (
	"runtime"
	"strings"

	"github.com/resonancelabs/go-pub/instrument"
	"golang.org/x/net/context"
)

// NewContext starts a new span named after the calling function
// and returns it in a new context.
// The span's operation will be named after the caller.
// If ctx already contains a span,
// the new span will set the old span as its parent,
// and it will use the old span's join IDs.
// Otherwise the new span will generate a new join ID.
func NewContext(ctx context.Context) context.Context {
	return newContextSuffix(ctx, "")
}

// NewContextSuffix is like NewContext but it adds suffix
// to the operation name.
func NewContextSuffix(ctx context.Context, suffix string) context.Context {
	return newContextSuffix(ctx, suffix)
}

func newContextSuffix(ctx context.Context, suffix string) context.Context {
	var pc [1]uintptr
	runtime.Callers(3, pc[:])
	name := funcName(pc[0])
	return NewContextName(ctx, name+suffix)
}

// NewContextName starts a new span for the named operation
// and returns it in a new context.
// If ctx already contains a span,
// the new span will set the old span as its parent,
// and it will use the old span's join IDs.
// Otherwise the new span will generate a new join ID.
func NewContextName(ctx context.Context, operation string) context.Context {
	sp := instrument.StartSpan()
	sp.SetOperation(operation)
	if parent := fromContext(ctx); parent != nil {
		sp.SetParent(parent)
	} else {
		sp.AddTraceJoinId("rootguid", sp.Guid()) // this is the root
	}
	ctx = NewContextWithSpan(ctx, sp)
	return ctx
}

func funcName(pc uintptr) string {
	f := runtime.FuncForPC(pc)
	if f == nil {
		return "unknown"
	}
	s := f.Name()
	return strings.Map(nameMap, s[strings.LastIndex(s, "/")+1:])
}

// remove unwanted characters
func nameMap(r rune) rune {
	if r == '(' || r == ')' || r == '*' {
		return -1
	}
	return r
}

// Finish finishes the current span
// and records the elapsed time in a histogram.
func Finish(ctx context.Context) {
	fromContext(ctx).Finish()
}

// LoggerFromContext returns the Logger stored in ctx, if any.
func LoggerFromContext(ctx context.Context) instrument.Logger {
	a := fromContext(ctx)
	if a == nil { // must avoid nil-object -> nonnil-interface
		return nil
	}
	return a
}
