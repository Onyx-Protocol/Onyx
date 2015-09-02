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

// Wrap adds additional context to err and returns a new error with the
// new context. Arguments are handled as in fmt.Print.
// Use Root to recover the original error wrapped by one or more calls to Wrap.
// Wrap returns nil if err is nil.
func Wrap(err error, a ...interface{}) error {
	if err == nil {
		return nil
	}

	werr, ok := err.(wrapperError)
	if !ok {
		werr.root = err
		werr.msg = err.Error()
	}
	if len(a) > 0 {
		werr.msg = fmt.Sprint(a...) + ": " + werr.msg
	}
	return werr
}

// Wrapf adds additional context to err and returns a new error with the
// new context. Arguments are handled as in fmt.Print.
// Use Root to recover the original error wrapped by one or more calls to Wrap.
// Wrapf returns nil if err is nil.
func Wrapf(err error, format string, a ...interface{}) error {
	return Wrap(err, fmt.Sprintf(format, a...))
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
	e1 := Wrap(err, text).(wrapperError)
	e1.detail = append(e1.detail, text)
	return e1
}

// WithDetailf is like WithDetail, except it formats
// the detail message as in fmt.Printf.
// Function Detail will return the formatted text
// when called on the new error value.
func WithDetailf(err error, format string, v ...interface{}) error {
	return WithDetail(err, fmt.Sprintf(format, v...))
}

// Detail returns the detail message contained in err, if any.
// An error has a detail message if it was made by WithDetail
// or WithDetailf.
func Detail(err error) string {
	wrapper, _ := err.(wrapperError)
	return strings.Join(wrapper.detail, "; ")
}
