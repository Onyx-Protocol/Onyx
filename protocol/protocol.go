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
blockchain state (using GetTxs or GetBlock) until you find that
the transaction has been either confirmed or rejected.

To ingest a block, call ValidateBlock and CommitBlock.

*/
package protocol

import (
	"context"
	"sync"

	"chain/crypto/ed25519"
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
	GetTxs(context.Context, ...bc.Hash) (bcTxs map[bc.Hash]*bc.Tx, err error)
	GetBlock(context.Context, uint64) (*bc.Block, error)
	LatestSnapshot(context.Context) (*state.Snapshot, uint64, error)

	SaveBlock(context.Context, *bc.Block) error
	FinalizeBlock(context.Context, uint64) error
	SaveSnapshot(context.Context, uint64, *state.Snapshot) error
}

// Pool provides storage for transactions in the pending tx pool.
type Pool interface {
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
	trustedKeys    []ed25519.PublicKey
	height         struct {
		cond sync.Cond // protects n
		n    uint64
	}
	store Store
	pool  Pool
}

// NewChain returns a new Chain using store as the underlying storage.
//
// ValidateBlock will skip validation for any block signed by a key
// in trustedKeys. Typically, trustedKeys contains the public key
// for the local block-signer process; the presence of its
// signature indicates the block was already validated locally.
func NewChain(ctx context.Context, store Store, pool Pool, trustedKeys []ed25519.PublicKey, heights <-chan uint64) (*Chain, error) {
	c := &Chain{
		store:       store,
		pool:        pool,
		trustedKeys: trustedKeys,
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

// Reset resets the blockchain height back to 0. It does not
// modify the Store.
func (c *Chain) Reset() {
	c.height.cond.L.Lock()
	defer c.height.cond.L.Unlock()
	c.height.n = 0
}

func (c *Chain) AddBlockCallback(f BlockCallback) {
	c.blockCallbacks = append(c.blockCallbacks, f)
}

func (c *Chain) WaitForBlock(ctx context.Context, height uint64) error {
	const slop = 3

	c.height.cond.L.Lock()
	defer c.height.cond.L.Unlock()

	if height > c.height.n+slop {
		return ErrTheDistantFuture
	}

	for c.height.n < height {
		c.height.cond.Wait()
	}

	return nil
}

// PendingTxs looks up the provided hashes in the tx pool.
func (c *Chain) PendingTxs(ctx context.Context, hashes ...bc.Hash) (map[bc.Hash]*bc.Tx, error) {
	return c.pool.GetTxs(ctx, hashes...)
}
