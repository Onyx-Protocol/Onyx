package sha3pool

import (
	"sync"

	"golang.org/x/crypto/sha3"
)

var pool = &sync.Pool{New: func() interface{} { return sha3.New256() }}

func Get() sha3.ShakeHash {
	return pool.Get().(sha3.ShakeHash)
}

func Put(h sha3.ShakeHash) {
	h.Reset()
	pool.Put(h)
}
