package errors

import "io"

// NewWriter returns a new Writer that writes to w
// until an error is returned.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Writer is in an implementation of the
// "sticky error writer" pattern as described
// in https://blog.golang.org/errors-are-values.
//
// A Writer makes one call
// on the underlying writer for each call to Write,
// until an error is returned. From that point on,
// it makes no calls on the underlying writer,
// and returns the same error value every time.
type Writer struct {
	w   io.Writer
	n   int64
	err error
}

// Write makes one call on the underlying writer
// if no error has previously occurred.
func (w *Writer) Write(buf []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	n, w.err = w.w.Write(buf)
	w.n += int64(n)
	return n, w.err
}

// Err returns the first error encountered by Write, if any.
func (w *Writer) Err() error {
	return w.err
}

// Written returns the number of bytes written
// to the underlying writer.
func (w *Writer) Written() int64 {
	return w.n
}
