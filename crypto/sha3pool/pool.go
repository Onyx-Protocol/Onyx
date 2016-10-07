// Package sha3pool is a freelist for SHA3-256 hash objects.
package sha3pool

import (
	"sync"

	"golang.org/x/crypto/sha3"
)

var pool = &sync.Pool{New: func() interface{} { return sha3.New256() }}

// Get256 returns an initialized SHA3-256 hash ready to use.
// It is like sha3.New256 except it uses the freelist.
// The caller should call Put256 when finished with the returned object.
func Get256() sha3.ShakeHash {
	return pool.Get().(sha3.ShakeHash)
}

// Put256 resets h and puts it in the freelist.
func Put256(h sha3.ShakeHash) {
	h.Reset()
	pool.Put(h)
}

// Sum256 uses a ShakeHash from the pool to sum into hash.
func Sum256(hash, data []byte) {
	h := Get256()
	h.Write(data)
	h.Read(hash)
	Put256(h)
}
