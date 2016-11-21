// Package log implements a standard convention for structured logging.
// Log entries are formatted as K=V pairs.
// By default, output is written to stdout; this can be changed with SetOutput.
package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"chain/errors"
	"chain/net/http/reqid"
)

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

var (
	logWriterMu sync.Mutex // protects the following
	logWriter   io.Writer  = os.Stdout
	prefix      []byte

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
	KeyCaller   = "at"       // location of caller
	KeyTime     = "t"        // time of call
	KeyReqID    = "reqid"    // request ID from context
	KeyCoreID   = "coreid"   // core ID from context
	KeySubReqID = "subreqid" // potential sub-request ID from context

	KeyMessage = "message" // produced by Message
	KeyError   = "error"   // produced by Error
	KeyStack   = "stack"   // used by Write to print stack on subsequent lines

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

// SetPrefix sets the output prefix.
func SetPrefix(keyval ...interface{}) {
	// Invariant: len(keyval) is always even.
	if len(keyval)%2 != 0 {
		panic(fmt.Sprintf("odd-length prefix args: %v", keyval))
	}
	var b []byte
	for i := 0; i < len(keyval); i += 2 {
		k := formatKey(keyval[i])
		v := formatValue(keyval[i+1])
		b = append(b, k...)
		b = append(b, '=')
		b = append(b, v...)
		b = append(b, ' ')
	}
	logWriterMu.Lock()
	prefix = b
	logWriterMu.Unlock()
}

// Write writes a structured log entry to stdout. Log fields are
// specified as a variadic sequence of alternating keys and values.
//
// Duplicate keys will be preserved.
//
// Several fields are automatically added to the log entry: a timestamp, a
// string indicating the file and line number of the caller, and a request ID
// taken from the context.
//
// As a special case, the auto-generated caller may be overridden by passing in
// a new value for the KeyCaller key as the first key-value pair. The override
// feature should be reserved for custom logging functions that wrap Write.
//
// Write will also print the stack trace, if any, on separate lines
// following the message. The stack is obtained from the following,
// in order of preference:
//   - a KeyStack value with type []byte or []errors.StackFrame
//   - a KeyError value with type error, using the result of errors.Stack
func Write(ctx context.Context, keyvals ...interface{}) {
	// Invariant: len(keyvals) is always even.
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "", keyLogError, "odd number of log params")
	}

	// The auto-generated caller value may be overwritten.
	var vcaller string
	if len(keyvals) >= 2 && keyvals[0] == KeyCaller {
		vcaller = formatValue(keyvals[1])
		keyvals = keyvals[2:]
	} else {
		vcaller = caller(1)
	}

	t := time.Now().UTC()

	// Prepend the log entry with auto-generated fields.
	out := fmt.Sprintf(
		"%s=%s %s=%s %s=%s",
		KeyReqID, formatValue(reqid.FromContext(ctx)),
		KeyCaller, vcaller,
		KeyTime, formatValue(t.Format(rfc3339NanoFixed)),
	)
	if s := reqid.CoreIDFromContext(ctx); s != "" {
		out += " " + KeyCoreID + "=" + formatValue(reqid.CoreIDFromContext(ctx))
	}

	if subreqid := reqid.FromSubContext(ctx); subreqid != reqid.Unknown {
		out += " " + KeySubReqID + "=" + formatValue(subreqid)
	}

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
	logWriter.Write(prefix)
	logWriter.Write([]byte(out)) // ignore errors
	logWriter.Write([]byte{'\n'})
	writeRawStack(logWriter, stack)
	logWriterMu.Unlock()
}

// Fatal is equivalent to Write() followed by a call to os.Exit(1).
func Fatal(ctx context.Context, keyvals ...interface{}) {
	Write(ctx, keyvals...)
	os.Exit(1)
}

func writeRawStack(w io.Writer, v interface{}) {
	switch v := v.(type) {
	case []byte:
		if len(v) > 0 {
			w.Write(v)
			w.Write([]byte{'\n'})
		}
	case []errors.StackFrame:
		for _, s := range v {
			io.WriteString(w, s.String())
			w.Write([]byte{'\n'})
		}
	}
}

func isStackVal(v interface{}) bool {
	switch v.(type) {
	case []byte:
		return true
	case []errors.StackFrame:
		return true
	}
	return false
}

// Messagef writes a log entry containing a message assigned to the
// "message" key. Arguments are handled as in fmt.Printf.
func Messagef(ctx context.Context, format string, a ...interface{}) {
	Write(ctx, KeyCaller, caller(1), KeyMessage, fmt.Sprintf(format, a...))
}

// Error writes a log entry containing an error message assigned to the
// "error" key.
// Optionally, an error message prefix can be included. Prefix arguments are
// handled as in fmt.Print.
func Error(ctx context.Context, err error, a ...interface{}) {
	if len(a) > 0 && len(errors.Stack(err)) > 0 {
		err = errors.Wrap(err, a...) // keep err's stack
	} else if len(a) > 0 {
		err = fmt.Errorf("%s: %s", fmt.Sprint(a...), err) // don't add a stack here
	}
	Write(ctx, KeyCaller, caller(1), KeyError, err)
}

// caller returns a string containing filename and line number of a
// function invocation on the calling goroutine's stack.
// The argument skip is the number of stack frames to ascend, where
// 0 is the calling site of caller. If no stack information is not available,
// "?:?" is returned.
func caller(skip int) string {
	_, file, nline, ok := runtime.Caller(skip + 1)

	var line string
	if ok {
		file = filepath.Base(file)
		line = strconv.Itoa(nline)
	} else {
		file = "?"
		line = "?"
	}

	return file + ":" + line
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
		Write(ctx,
			KeyMessage, "panic",
			KeyError, err,
			KeyStack, buf,
		)
	}
}
