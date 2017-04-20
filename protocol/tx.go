package protocol

import (
	"sync"

	"github.com/golang/groupcache/lru"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/validation"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// ValidateTx validates the given transaction. A cache holds
// per-transaction validation results and is consulted before
// performing full validation.
func (c *Chain) ValidateTx(tx *bc.Tx) error {
	err := c.checkIssuanceWindow(tx)
	if err != nil {
		return err
	}
	var ok bool
	err, ok = c.prevalidated.lookup(tx.ID)
	if !ok {
		err = validation.ValidateTx(tx, c.InitialBlockHash)
		c.prevalidated.cache(tx.ID, err)
	}
	return errors.Sub(ErrBadTx, err)
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

func (c *Chain) checkIssuanceWindow(tx *bc.Tx) error {
	if c.MaxIssuanceWindow == 0 {
		return nil
	}
	for _, entryID := range tx.InputIDs {
		if _, err := tx.Issuance(entryID); err == nil {
			if tx.MinTimeMs+bc.DurationMillis(c.MaxIssuanceWindow) < tx.MaxTimeMs {
				return errors.WithDetailf(ErrBadTx, "issuance input's time window is larger than the network maximum (%s)", c.MaxIssuanceWindow)
			}
		}
	}
	return nil
}
