package protocol

import (
	"sync"

	"github.com/golang/groupcache/lru"

	"chain/protocol/bc"
)

// maxCachedValidatedTxs is the max number of validated txs to cache.
const maxCachedValidatedTxs = 1000

type prevalidatedTxsCache struct {
	mu  sync.Mutex
	lru *lru.Cache
}

func newPrevalidatedTxsCache() *prevalidatedTxsCache {
	return &prevalidatedTxsCache{
		lru: lru.New(maxCachedValidatedTxs),
	}
}

func (c *prevalidatedTxsCache) lookup(txID bc.Hash) (err error, ok bool) {
	c.mu.Lock()
	v, ok := c.lru.Get(txID)
	c.mu.Unlock()
	if !ok {
		return nil, false
	}
	if v == nil {
		return nil, true
	}
	return v.(error), true
}

func (c *prevalidatedTxsCache) cache(txID bc.Hash, err error) {
	c.mu.Lock()
	c.lru.Add(txID, err)
	c.mu.Unlock()
}
