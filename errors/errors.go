package errors

import (
	"errors"
	"fmt"
	"strings"
)

// New returns an error that formats as the given text.
func New(text string) error {
	return errors.New(text)
}

// wrapperError satisfies the error interface.
type wrapperError struct {
	msg    string
	detail []string
	data   map[string]interface{}
	stack  []StackFrame
	root   error
}

// It satisfies the error interface.
func (e wrapperError) Error() string {
	return e.msg
}

// Root returns the original error that was wrapped by one or more
// calls to Wrap. If e does not wrap other errors, it will be returned
// as-is.
func Root(e error) error {
	if wErr, ok := e.(wrapperError); ok {
		return wErr.root
	}
	return e
}

// wrap adds a context message and stack trace to err and returns a new error
// containing the new context. This function is meant to be composed within
// other exported functions, such as Wrap and WithDetail.
// The argument stackSkip is the number of stack frames to ascend when
// generating stack straces, where 0 is the caller of wrap.
func wrap(err error, msg string, stackSkip int) error {
	if err == nil {
		return nil
	}

	werr, ok := err.(wrapperError)
	if !ok {
		werr.root = err
		werr.msg = err.Error()
		werr.stack = getStack(stackSkip+2, stackTraceSize)
	}
	if msg != "" {
		werr.msg = msg + ": " + werr.msg
	}

	return werr
}

// Wrap adds a context message and stack trace to err and returns a new error
// with the new context. Arguments are handled as in fmt.Print.
// Use Root to recover the original error wrapped by one or more calls to Wrap.
// Use Stack to recover the stack trace.
// Wrap returns nil if err is nil.
func Wrap(err error, a ...interface{}) error {
	return wrap(err, fmt.Sprint(a...), 1)
}

// Wrapf is like Wrap, but arguments are handled as in fmt.Printf.
func Wrapf(err error, format string, a ...interface{}) error {
	return wrap(err, fmt.Sprintf(format, a...), 1)
}

// WithDetail returns a new error that wraps
// err as a chain error messsage containing text
// as its additional context.
// Function Detail will return the given text
// when called on the new error value.
func WithDetail(err error, text string) error {
	if err == nil {
		return nil
	}
	if text == "" {
		return err
	}
	e1 := wrap(err, text, 1).(wrapperError)
	e1.detail = append(e1.detail, text)
	return e1
}

// WithDetailf is like WithDetail, except it formats
// the detail message as in fmt.Printf.
// Function Detail will return the formatted text
// when called on the new error value.
func WithDetailf(err error, format string, v ...interface{}) error {
	if err == nil {
		return nil
	}
	text := fmt.Sprintf(format, v...)
	e1 := wrap(err, text, 1).(wrapperError)
	e1.detail = append(e1.detail, text)
	return e1
}

// Detail returns the detail message contained in err, if any.
// An error has a detail message if it was made by WithDetail
// or WithDetailf.
func Detail(err error) string {
	wrapper, _ := err.(wrapperError)
	return strings.Join(wrapper.detail, "; ")
}

// withData returns a new error that wraps err
// as a chain error message containing v as
// an extra data item.
// Calling Data on the returned error yields v.
// Note that if err already has a data item,
// it will not be accessible via the returned error value.
func withData(err error, v map[string]interface{}) error {
	if err == nil {
		return nil
	}
	e1 := wrap(err, "", 1).(wrapperError)
	e1.data = v
	return e1
}

// WithData returns a new error that wraps err
// as a chain error message containing a value of type
// map[string]interface{} as an extra data item.
// The map contains the values in the map in err,
// if any, plus the items in keyval.
// Keyval takes the form
//   k1, v1, k2, v2, ...
// Values kN must be strings.
// Calling Data on the returned error yields the map.
// Note that if err already has a data item of any other type,
// it will not be accessible via the returned error value.
func WithData(err error, keyval ...interface{}) error {
	// TODO(kr): add vet check for odd-length keyval and non-string keys
	newkv := make(map[string]interface{})
	for k, v := range Data(err) {
		newkv[k] = v
	}
	for i := 0; i < len(keyval); i += 2 {
		newkv[keyval[i].(string)] = keyval[i+1]
	}
	return withData(err, newkv)
}

// Data returns the data item in err, if any.
func Data(err error) map[string]interface{} {
	wrapper, _ := err.(wrapperError)
	return wrapper.data
}
