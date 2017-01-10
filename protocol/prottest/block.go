package prottest

import (
	"context"
	"sync"
	"testing"
	"time"

	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/memstore"
	"chain/protocol/state"
	"chain/testutil"
)

var (
	mutex  sync.Mutex // protects the following
	states = make(map[*protocol.Chain]*state.Snapshot)
)

// NewChain makes a new Chain using memstore for storage,
// along with an initial block using a 0/0 multisig program.
// It commits the initial block before returning the Chain.
func NewChain(tb testing.TB) *protocol.Chain {
	return NewChainWithStorage(tb, memstore.New())
}

// NewChainWithStorage makes a new Chain using store for storage, along
// with an initial block using a 0/0 multisig program.
// It commits the initial block before returning the Chain.
func NewChainWithStorage(tb testing.TB, store protocol.Store) *protocol.Chain {
	ctx := context.Background()
	b1, err := protocol.NewInitialBlock(nil, 0, time.Now())
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	c, err := protocol.NewChain(ctx, b1.Hash(), store, nil)
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

// MakeBlock makes a new block from txs, commits it, and returns it.
// It assumes c's consensus program requires 0 signatures.
// (This is true for chains returned by NewChain.)
// If c requires more than 0 signatures, MakeBlock will fail.
// MakeBlock always makes a block;
// if there are no transactions in txs,
// it makes an empty block.
func MakeBlock(tb testing.TB, c *protocol.Chain, txs []*bc.Tx) *bc.Block {
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

	nextBlock, nextState, err := c.GenerateBlock(ctx, curBlock, curState, time.Now(), txs)
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
