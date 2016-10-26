package prottest

import (
	"context"
	"sync"
	"testing"
	"time"

	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/mempool"
	"chain/protocol/memstore"
	"chain/protocol/state"
	"chain/testutil"
)

var (
	mutex  sync.Mutex // protects the following
	states = make(map[*protocol.Chain]*state.Snapshot)
)

// NewChain makes a new Chain using memstore and mempool for storage,
// along with an initial block using a 0/0 multisig program.
// It commits the initial block before returning the Chain.
func NewChain(tb testing.TB) *protocol.Chain {
	return NewChainWithStorage(tb, memstore.New(), mempool.New())
}

// NewChainWithStorage makes a new Chain using store and pool for
// storage, along with an initial block using a 0/0 multisig program.
// It commits the initial block before returning the Chain.
func NewChainWithStorage(tb testing.TB, store protocol.Store, pool protocol.Pool) *protocol.Chain {
	ctx := context.Background()
	b1, err := protocol.NewInitialBlock(nil, 0, time.Now())
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	c, err := protocol.NewChain(ctx, b1.Hash(), store, pool, nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	c.MaxIssuanceWindow = 48 * time.Hour // TODO(tessr): consider adding MaxIssuanceWindow to NewChain
	err = c.CommitBlock(ctx, b1, state.Empty())
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	return c
}

// MakeBlock makes a new block from the pool in c, commits it, and returns it.
// It assumes c's consensus program requires 0 signatures.
// (This is true for chains returned by NewChain.)
// If c requires more than 0 signatures, MakeBlock will fail.
// MakeBlock always makes a block;
// if there are no transactions in the pool,
// it makes an empty block.
func MakeBlock(tb testing.TB, c *protocol.Chain) *bc.Block {
	ctx := context.Background()
	curBlock, err := c.GetBlock(ctx, c.Height())
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	mutex.Lock()
	curState := states[c]
	mutex.Unlock()
	if curState == nil {
		curState = state.Empty()
	}

	nextBlock, nextState, err := c.GenerateBlock(ctx, curBlock, curState, time.Now())
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	err = c.CommitBlock(ctx, nextBlock, nextState)
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	mutex.Lock()
	states[c] = nextState
	mutex.Unlock()
	return nextBlock
}
