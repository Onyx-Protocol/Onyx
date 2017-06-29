package prottest

import (
	"context"
	"sync"
	"testing"
	"time"

	"chain/crypto/ed25519"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/bc/bcvm"
	"chain/protocol/prottest/memstore"
	"chain/protocol/state"
	"chain/testutil"
)

var (
	mutex         sync.Mutex // protects the following
	states        = make(map[*protocol.Chain]*state.Snapshot)
	blockPubkeys  = make(map[*protocol.Chain][]ed25519.PublicKey)
	blockPrivkeys = make(map[*protocol.Chain][]ed25519.PrivateKey)
)

type Option func(testing.TB, *config)

func WithStore(store protocol.Store) Option {
	return func(_ testing.TB, conf *config) { conf.store = store }
}

func WithOutputIDs(outputIDs ...bc.Hash) Option {
	return func(_ testing.TB, conf *config) {
		for _, oid := range outputIDs {
			conf.initialState.Tree.Insert(oid.Bytes())
		}
	}
}

func WithBlockSigners(quorum, n int) Option {
	return func(tb testing.TB, conf *config) {
		conf.quorum = quorum
		for i := 0; i < n; i++ {
			pubkey, privkey, err := ed25519.GenerateKey(nil)
			if err != nil {
				testutil.FatalErr(tb, err)
			}
			conf.pubkeys = append(conf.pubkeys, pubkey)
			conf.privkeys = append(conf.privkeys, privkey)
		}
	}
}

type config struct {
	store        protocol.Store
	initialState *state.Snapshot
	pubkeys      []ed25519.PublicKey
	privkeys     []ed25519.PrivateKey
	quorum       int
}

// NewChain makes a new Chain. By default it uses a memstore for
// storage and creates an initial block using a 0/0 multisig program.
// It commits the initial block before returning the Chain.
//
// Its defaults may be overridden by providing Options.
func NewChain(tb testing.TB, opts ...Option) *protocol.Chain {
	conf := config{store: memstore.New(), initialState: state.Empty()}
	for _, opt := range opts {
		opt(tb, &conf)
	}

	ctx := context.Background()
	b1, err := protocol.NewInitialBlock(conf.pubkeys, conf.quorum, time.Now())
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	c, err := protocol.NewChain(ctx, b1.Hash(), conf.store, nil)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	c.MaxIssuanceWindow = 48 * time.Hour // TODO(tessr): consider adding MaxIssuanceWindow to NewChain

	err = c.CommitAppliedBlock(ctx, b1, conf.initialState)
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	// save block-signing keys in global state
	mutex.Lock()
	blockPubkeys[c] = conf.pubkeys
	blockPrivkeys[c] = conf.privkeys
	mutex.Unlock()

	return c
}

// Initial returns the provided Chain's initial block.
func Initial(tb testing.TB, c *protocol.Chain) *bcvm.Block {
	ctx := context.Background()
	b1, err := c.GetBlock(ctx, 1)
	if err != nil {
		testutil.FatalErr(tb, err)
	}
	return b1
}

// BlockKeyPairs returns the configured block-signing key-pairs
// for the provided Chain.
func BlockKeyPairs(c *protocol.Chain) ([]ed25519.PublicKey, []ed25519.PrivateKey) {
	mutex.Lock()
	defer mutex.Unlock()
	return blockPubkeys[c], blockPrivkeys[c]
}

// MakeBlock makes a new block from txs, commits it, and returns it.
// It assumes c's consensus program requires 0 signatures.
// (This is true for chains returned by NewChain.)
// If c requires more than 0 signatures, MakeBlock will fail.
// MakeBlock always makes a block;
// if there are no transactions in txs,
// it makes an empty block.
func MakeBlock(tb testing.TB, c *protocol.Chain, txs [][]byte) *bcvm.Block {
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
	err = c.CommitAppliedBlock(ctx, nextBlock, nextState)
	if err != nil {
		testutil.FatalErr(tb, err)
	}

	mutex.Lock()
	states[c] = nextState
	mutex.Unlock()
	return nextBlock
}
