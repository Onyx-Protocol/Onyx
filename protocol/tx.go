package protocol

import (
	"context"
	"sync"

	"github.com/golang/groupcache/lru"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/validation"
)

// AddTx inserts tx into the set of "pending" transactions available
// to be included in the next block produced by GenerateBlock. It should only
// be called by the Generator.
//
// It performs context-free validation of the tx, but does not validate
// against the current state tree.
//
// It is okay to add the same transaction more than once; subsequent
// attempts will have no effect and return a nil error.
//
// It is okay to add conflicting transactions to the pool. The conflict
// will be resolved when a block lands.
//
// It is an error to call AddTx before the initial block has landed.
// Use WaitForBlock to guarantee this.
func (c *Chain) AddTx(ctx context.Context, tx *bc.Tx) error {
	err := c.ValidateTxCached(tx)
	if err != nil {
		return errors.Wrap(err, "tx rejected")
	}

	// Update persistent tx pool state.
	err = c.pool.Insert(ctx, tx)
	return errors.Wrap(err, "applying tx to store")
}

// ValidateTxCached checks a cache of prevalidated transactions
// before attempting to perform a context-free validation of the tx.
func (c *Chain) ValidateTxCached(tx *bc.Tx) error {
	// Consult a cache of prevalidated transactions.
	err, ok := c.prevalidated.lookup(tx.Hash)
	if ok {
		return err
	}

	err = validation.ValidateTx(tx)
	c.prevalidated.cache(tx.Hash, err)
	return err
}

type prevalidatedTxsCache struct {
	mu  sync.Mutex
	lru *lru.Cache
}

func (c *prevalidatedTxsCache) lookup(txID bc.Hash) (err error, ok bool) {
	c.mu.Lock()
	v, ok := c.lru.Get(txID)
	c.mu.Unlock()
	if !ok {
		return err, ok
	}
	if v == nil {
		return nil, ok
	}
	return v.(error), ok
}

func (c *prevalidatedTxsCache) cache(txID bc.Hash, err error) {
	c.mu.Lock()
	c.lru.Add(txID, err)
	c.mu.Unlock()
}
