package protocol

import (
	"context"

	"chain/errors"
	"chain/protocol/bc"
)

// AddTx inserts tx into the set of "pending" transactions available
// to be included in the next block produced by GenerateBlock.
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
	err := c.validateTxCached(tx)
	if err != nil {
		return errors.Wrap(err, "tx rejected")
	}

	// Update persistent tx pool state.
	err = c.pool.Insert(ctx, tx)
	return errors.Wrap(err, "applying tx to store")
}
