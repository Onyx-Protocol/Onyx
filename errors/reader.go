package errors

import "io"

// NewReader returns a new Reader that reads from r
// until an error is returned.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// Reader is in an implementation of the
// "sticky error" pattern as described
// in https://blog.golang.org/errors-are-values.
//
// A Reader makes one call
// on the underlying reader for each call to Read,
// until an error is returned. From that point on,
// it makes no calls on the underlying reader,
// and returns the same error value every time.
type Reader struct {
	r   io.Reader
	n   int64
	err error
}

// Read makes one call on the underlying reader
// if no error has previously occurred.
func (r *Reader) Read(buf []byte) (n int, err error) {
	if r.err != nil {
		return 0, r.err
	}
	n, r.err = r.r.Read(buf)
	r.n += int64(n)
	return n, r.err
}

// Err returns the first error encountered by Read, if any.
func (r *Reader) Err() error {
	return r.err
}

// BytesRead returns the number of bytes read
// from the underlying reader.
func (r *Reader) BytesRead() int64 {
	return r.n
}
