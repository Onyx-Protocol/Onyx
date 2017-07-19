// Package log implements a standard convention for structured logging.
// Log entries are formatted as K=V pairs.
// By default, output is written to stdout; this can be changed with SetOutput.
package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"chain/errors"
)

// context key type
type key int

var (
	logWriterMu sync.Mutex // protects the following
	logWriter   io.Writer  = os.Stdout
	procPrefix  []byte     // process-global prefix; see SetPrefix vs AddPrefixkv

	// context key for log line prefixes
	prefixKey key = 0
)

const (
	rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

	// pairDelims contains a list of characters that may be used as delimeters
	// between key-value pairs in a log entry. Keys and values will be quoted or
	// otherwise formatted to ensure that key-value extraction is unambiguous.
	//
	// The list of pair delimiters follows Splunk conventions, described here:
	// http://answers.splunk.com/answers/143368/default-delimiters-for-key-value-extraction.html
	pairDelims      = " ,;|&\t\n\r"
	illegalKeyChars = pairDelims + `="`
)

// Conventional key names for log entries
const (
	KeyCaller = "at" // location of caller
	KeyTime   = "t"  // time of call

	KeyMessage = "message" // produced by Message
	KeyError   = "error"   // produced by Error
	KeyStack   = "stack"   // used by Printkv to print stack on subsequent lines

	keyLogError = "log-error" // for errors produced by the log package itself
)

// SetOutput sets the log output to w.
// If SetOutput hasn't been called,
// the default behavior is to write to stdout.
func SetOutput(w io.Writer) {
	logWriterMu.Lock()
	logWriter = w
	logWriterMu.Unlock()
}

func appendPrefix(b []byte, keyval ...interface{}) []byte {
	// Invariant: len(keyval) is always even.
	if len(keyval)%2 != 0 {
		panic(fmt.Sprintf("odd-length prefix args: %v", keyval))
	}
	for i := 0; i < len(keyval); i += 2 {
		k := formatKey(keyval[i])
		v := formatValue(keyval[i+1])
		b = append(b, k...)
		b = append(b, '=')
		b = append(b, v...)
		b = append(b, ' ')
	}
	return b
}

// SetPrefix sets the global output prefix.
func SetPrefix(keyval ...interface{}) {
	b := appendPrefix(nil, keyval...)
	logWriterMu.Lock()
	procPrefix = b
	logWriterMu.Unlock()
}

// AddPrefixkv appends keyval to any prefix stored in ctx,
// and returns a new context with the longer prefix.
func AddPrefixkv(ctx context.Context, keyval ...interface{}) context.Context {
	p := appendPrefix(prefix(ctx), keyval...)
	// Note: subsequent calls will append to p, so set cap(p) here.
	// See TestAddPrefixkvAppendTwice.
	p = p[0:len(p):len(p)]
	return context.WithValue(ctx, prefixKey, p)
}

func prefix(ctx context.Context) []byte {
	b, _ := ctx.Value(prefixKey).([]byte)
	return b
}

// Printkv prints a structured log entry to stdout. Log fields are
// specified as a variadic sequence of alternating keys and values.
//
// Duplicate keys will be preserved.
//
// Two fields are automatically added to the log entry: t=[time]
// and at=[file:line] indicating the location of the caller.
// Use SkipFunc to prevent helper functions from showing up in the
// at=[file:line] field.
//
// Printkv will also print the stack trace, if any, on separate lines
// following the message. The stack is obtained from the following,
// in order of preference:
//   - a KeyStack value with type []byte or *runtime.Frames
//   - a KeyError value with type error, using the result of errors.Stack
func Printkv(ctx context.Context, keyvals ...interface{}) {
	// Invariant: len(keyvals) is always even.
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "", keyLogError, "odd number of log params")
	}

	t := time.Now().UTC()

	// Prepend the log entry with auto-generated fields.
	out := fmt.Sprintf(
		"%s=%s %s=%s",
		KeyCaller, caller(),
		KeyTime, formatValue(t.Format(rfc3339NanoFixed)),
	)

	var stack interface{}
	for i := 0; i < len(keyvals); i += 2 {
		k := keyvals[i]
		v := keyvals[i+1]
		if k == KeyStack && isStackVal(v) {
			stack = v
			continue
		}
		if k == KeyError {
			if e, ok := v.(error); ok && stack == nil {
				stack = errors.Stack(errors.Wrap(e)) // wrap to ensure callstack
			}
		}
		out += " " + formatKey(k) + "=" + formatValue(v)
	}

	logWriterMu.Lock()
	logWriter.Write(procPrefix)
	logWriter.Write(prefix(ctx))
	logWriter.Write([]byte(out)) // ignore errors
	logWriter.Write([]byte{'\n'})
	writeRawStack(logWriter, stack)
	logWriterMu.Unlock()
}

// Fatalkv is equivalent to Printkv() followed by a call to os.Exit(1).
func Fatalkv(ctx context.Context, keyvals ...interface{}) {
	Printkv(ctx, keyvals...)
	os.Exit(1)
}

func writeRawStack(w io.Writer, v interface{}) {
	switch v := v.(type) {
	case []byte:
		if len(v) > 0 {
			w.Write(v)
			w.Write([]byte{'\n'})
		}
	case *runtime.Frames:
		for f, ok := v.Next(); ok; f, ok = v.Next() {
			fmt.Fprintf(w, "%s:%d: %s\n", f.File, f.Line, f.Function)
		}
	}
}

func isStackVal(v interface{}) bool {
	switch v.(type) {
	case []byte:
		return true
	case *runtime.Frames:
		return true
	}
	return false
}

// Printf prints a log entry containing a message assigned to the
// "message" key. Arguments are handled as in fmt.Printf.
func Printf(ctx context.Context, format string, a ...interface{}) {
	Printkv(ctx, KeyMessage, fmt.Sprintf(format, a...))
}

// Error prints a log entry containing an error message assigned to the
// "error" key.
// Optionally, an error message prefix can be included. Prefix arguments are
// handled as in fmt.Print.
func Error(ctx context.Context, err error, a ...interface{}) {
	if _, hasStack := errors.Stack(err).Next(); len(a) > 0 && hasStack {
		err = errors.Wrap(err, a...) // keep err's stack
	} else if len(a) > 0 {
		err = fmt.Errorf("%s: %s", fmt.Sprint(a...), err) // don't add a stack here
	}
	Printkv(ctx, KeyError, err)
}

// formatKey ensures that the stringified key is valid for use in a
// Splunk-style K=V format. It stubs out delimeter and quoter characters in
// the key string with hyphens.
func formatKey(k interface{}) string {
	s := fmt.Sprint(k)
	if s == "" {
		return "?"
	}

	for _, c := range illegalKeyChars {
		s = strings.Replace(s, string(c), "-", -1)
	}

	return s
}

// formatValue ensures that the stringified value is valid for use in a
// Splunk-style K=V format. It quotes the string value if delimeter or quoter
// characters are present in the value string.
func formatValue(v interface{}) string {
	s := fmt.Sprint(v)
	if strings.ContainsAny(s, pairDelims) {
		return strconv.Quote(s)
	}
	return s
}

// RecoverAndLogError must be used inside a defer.
func RecoverAndLogError(ctx context.Context) {
	if err := recover(); err != nil {
		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		Printkv(ctx,
			KeyMessage, "panic",
			KeyError, err,
			KeyStack, buf,
		)
	}
}
