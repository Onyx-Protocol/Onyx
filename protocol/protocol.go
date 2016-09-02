/*

Package protocol provides the core logic to tie together storage
and validation for a Chain Open Standard blockchain.
This comprises all behavior
that's common to every full node, as well as other functions that
need to operate on the blockchain state.

Here are a few examples of typical full node types.

Generator

A generator has two basic jobs: collecting transactions from
other nodes and putting them into blocks.

To collect pending transactions, call AddTx for each one.

To add a new block to the blockchain, call GenerateBlock, sign
the block (possibly collecting signatures from other parties),
and call CommitBlock.

Signer

An signer has one basic job: sign exactly one valid block at each height.

Manager

A manager's job is to select outputs for spending and to compose
transactions.

To publish a new transaction, prepare your transaction (select
outputs, and compose and sign the tx), call AddTx, and
send the transaction to a generator node. To wait for confirmation,
call WaitForBlock on successive block heights and inspect the
blockchain state until you find that
the transaction has been either confirmed or rejected.

To ingest a block, call ValidateBlock and CommitBlock.

*/
package protocol

import (
	"context"
	"sync"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/state"
)

var (
	// ErrTheDistantFuture (https://youtu.be/2IPAOxrH7Ro) is returned when
	// waiting for a blockheight too far in excess of the tip of the
	// blockchain.
	ErrTheDistantFuture = errors.New("the block height is too damn high")

	// ErrBadStateHeight is returned from Store.StateTree when the
	// height parameter does not match the latest block height.
	ErrBadStateHeight = errors.New("requested block height does not match current state")
)

type BlockCallback func(ctx context.Context, block *bc.Block)

// Store provides storage for blockchain data: blocks, asset
// definition pointers and confirmed transactions.
//
// Note, this is different from a state tree. A state tree provides
// access to the state at a given point in time -- outputs and
// ADPs. A Chain uses Store to load state trees from storage and
// persist validated data.
type Store interface {
	Height(context.Context) (uint64, error)
	GetBlock(context.Context, uint64) (*bc.Block, error)
	LatestSnapshot(context.Context) (*state.Snapshot, uint64, error)

	SaveBlock(context.Context, *bc.Block) error
	FinalizeBlock(context.Context, uint64) error
	SaveSnapshot(context.Context, uint64, *state.Snapshot) error
}

// Pool provides storage for transactions in the pending tx pool.
type Pool interface {
	// Insert adds a transaction to the pool.
	// It doesn't check for validity, or whether the transaction
	// conflicts with another.
	// It is required to be idempotent.
	Insert(context.Context, *bc.Tx) error

	GetTxs(context.Context, ...bc.Hash) (map[bc.Hash]*bc.Tx, error)
	Clean(ctx context.Context, txs []*bc.Tx) error
	Dump(context.Context) ([]*bc.Tx, error)
}

// Chain provides a complete, minimal blockchain database. It
// delegates the underlying storage to other objects, and uses
// validation logic from package validation to decide what
// objects can be safely stored.
type Chain struct {
	blockCallbacks []BlockCallback
	height         struct {
		cond sync.Cond // protects n
		n    uint64
	}
	store Store
	pool  Pool
}

// NewChain returns a new Chain using store as the underlying storage.
func NewChain(ctx context.Context, store Store, pool Pool, heights <-chan uint64) (*Chain, error) {
	c := &Chain{
		store: store,
		pool:  pool,
	}
	c.height.cond.L = new(sync.Mutex)

	var err error
	c.height.n, err = store.Height(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "looking up blockchain height")
	}

	// Note that c.height.n may still be zero here.
	if heights != nil {
		go func() {
			for h := range heights {
				c.setHeight(h)
			}
		}()
	}

	return c, nil
}

// Height returns the current height of the blockchain.
func (c *Chain) Height() uint64 {
	c.height.cond.L.Lock()
	defer c.height.cond.L.Unlock()
	return c.height.n
}

func (c *Chain) AddBlockCallback(f BlockCallback) {
	c.blockCallbacks = append(c.blockCallbacks, f)
}

// WaitForBlockSoon waits for the block at the given height,
// but it is an error to wait for a block far in the future.
// To wait unconditionally, the caller should use WaitForBlock.
func (c *Chain) WaitForBlockSoon(height uint64) error {
	const slop = 3
	if height > c.Height()+slop {
		return ErrTheDistantFuture
	}
	c.WaitForBlock(height)
	return nil
}

// WaitForBlock waits for the block at the given height.
func (c *Chain) WaitForBlock(height uint64) {
	c.height.cond.L.Lock()
	defer c.height.cond.L.Unlock()
	for c.height.n < height {
		c.height.cond.Wait()
	}
}

// PendingTxs looks up the provided hashes in the tx pool.
func (c *Chain) PendingTxs(ctx context.Context, hashes ...bc.Hash) (map[bc.Hash]*bc.Tx, error) {
	return c.pool.GetTxs(ctx, hashes...)
}
