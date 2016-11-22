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
func Get() *bytes.Buffer {
	return pool.Get().(*bytes.Buffer)
}

// Put resets the buffer and adds it to the freelist.
func Put(b *bytes.Buffer) {
	b.Reset()
	pool.Put(b)
}
