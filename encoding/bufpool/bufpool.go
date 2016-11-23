// Package bufpool is a freelist for bytes.Buffer objects.
package bufpool

import (
	"bytes"
	"sync"
)

var pool = &sync.Pool{New: func() interface{} { return bytes.NewBuffer(nil) }}

// GetBuffer returns an initialized bytes.Buffer object.
// It is like new(bytes.Buffer) except it uses the free list.
// The caller should call PutBuffer when finished with the returned object.
func GetBuffer() *bytes.Buffer {
	return pool.Get().(*bytes.Buffer)
}

// PutBuffer resets the buffer and adds it to the freelist.
func PutBuffer(b *bytes.Buffer) {
	b.Reset()
	pool.Put(b)
}
