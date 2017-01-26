// Package bufpool is a freelist for bytes.Buffer objects.
package bufpool

import (
	"bytes"
	"sync"
)

var pool = &sync.Pool{New: func() interface{} { return bytes.NewBuffer(nil) }}

// Get returns an initialized bytes.Buffer object.
// It is like new(bytes.Buffer) except it uses the free list.
// The caller should call Put when finished with the returned object.
// Since Buffer.Bytes() returns the buffer's underlying slice,
// it is not safe for that slice to escape the caller.
// If the bytes need to escape, CopyBytes should be used.
func Get() *bytes.Buffer {
	return pool.Get().(*bytes.Buffer)
}

// Put resets the buffer and adds it to the freelist.
func Put(b *bytes.Buffer) {
	b.Reset()
	pool.Put(b)
}

// CopyBytes returns a copy of the bytes contained in the buffer.
// This slice is safe from updates in the underlying buffer,
// allowing the buffer to be placed back in the free list.
func CopyBytes(buf *bytes.Buffer) []byte {
	b := buf.Bytes()
	b2 := make([]byte, len(b))
	copy(b2, b)
	return b2
}
