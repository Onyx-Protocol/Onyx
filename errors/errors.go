package errors

import "fmt"

// wrapperError satisfies the error interface.
type wrapperError struct {
	msg  string
	root error
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

	var root error

	if wErr, ok := err.(wrapperError); ok {
		root = wErr.root
	} else {
		root = err
	}

	return wrapperError{
		root: root,
		msg:  fmt.Sprint(a...) + ": " + err.Error(),
	}
}

// Wrapf adds additional context to err and returns a new error with the
// new context. Arguments are handled as in fmt.Print.
// Use Root to recover the original error wrapped by one or more calls to Wrap.
// Wrapf returns nil if err is nil.
func Wrapf(err error, format string, a ...interface{}) error {
	return Wrap(err, fmt.Sprintf(format, a...))
}
