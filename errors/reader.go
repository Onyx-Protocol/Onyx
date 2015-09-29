package errors

import "io"

// Reader is in an implementation of the
// "sticky error" pattern as described
// in https://blog.golang.org/errors-are-values.
//
// A Reader makes one call
// on the underlying reader for each call to Read,
// until an error is returned. From that point on,
// it makes no calls on the underlying reader,
// and returns the same error value every time.
//
// Each call to Read updates N
// to reflect the amount read so far.
// Each call to the underlying reader sets Err
// to the returned error.
type Reader struct {
	R   io.Reader
	N   int64
	Err error
}

// Read makes one call on the underlying reader
// if no error has previously occurred.
func (r *Reader) Read(buf []byte) (n int, err error) {
	if r.Err != nil {
		return 0, r.Err
	}
	n, r.Err = r.R.Read(buf)
	r.N += int64(n)
	return n, r.Err
}
