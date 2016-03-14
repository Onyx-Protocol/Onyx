/*

Package fedchain provides the core logic to tie together storage
and validation for a blockchain. This comprises all behavior
that's common to every full node, as well as other functions that
need to operate on the blockchain state.

Here are a few examples of typical full node types.

Generator

An generator has two basic jobs: collecting transactions from
other nodes and putting them into blocks.

To collect pending transactions, call InsertPendingTx for each one.

To add a new block to the blockchain, call GenerateBlock, sign
the block (possibly collecting signatures from other parties),
and call AddBlock.

Signer

An signer has one basic job: sign exactly one valid block at each height.

To sign an unsigned block obtained from a generator node, first
validate against the current blockchain state, then call
ComputeBlockSignature, and finally send the signature back to the
generator node.

Manager

A manager's job is to select outputs for spending and to compose
transactions.

To publish a new transaction, prepare your transaction (select
outputs, and compose and sign the tx), call InsertPendingTx and
send the transaction to a generator node.
(TODO(kr): Then call WaitTx; when it returns (with no error),
the transaction has been confirmed.)

To ingest a block, call AddBlock.

*/
package fedchain

import (
	"sync"

	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/btcec"

	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/patricia"
	"chain/fedchain/state"
)

var (
	// ErrTheDistantFuture (https://youtu.be/2IPAOxrH7Ro) is returned when
	// waiting for a blockheight too far in excess of the tip of the
	// blockchain.
	ErrTheDistantFuture = errors.New("the block height is too damn high")

	// ErrNoBlocks is returned when LatestBlock is called and the store
	// contains no blocks.
	ErrNoBlocks = errors.New("no blocks in the store")
)

type BlockCallback func(ctx context.Context, block *bc.Block, conflicts []*bc.Tx)
type TxCallback func(context.Context, *bc.Tx)

// Store provides storage for blockchain data: blocks, asset
// definition pointers, and pending transactions.
//
// Note, this is different from state.View. A View provides
// access to the state at a given point in time -- outputs and
// ADPs. It doesn't distinguish between blockchain outputs and
// pool outputs; a View instance might provide either, or both.
// Store, by contrast, provides lower-level access to the
// concrete blockchain. It can insert raw block data, add and
// remove pending txs from the pool, and update the UTXO state
// directly. An FC uses Store to create a set of Views,
// uses those views to validate a tx or block, then uses Store
// to commit the validated data.
type Store interface {
	// tx pool
	GetTxs(context.Context, ...bc.Hash) (map[bc.Hash]*bc.Tx, error)
	ApplyTx(context.Context, *bc.Tx, map[bc.AssetID]*state.AssetState) error
	CleanPool(ctx context.Context, confirmed, conflicting []*bc.Tx, assets map[bc.AssetID]*state.AssetState) error
	PoolTxs(context.Context) ([]*bc.Tx, error)
	NewPoolViewForPrevouts(context.Context, []*bc.Tx) (state.ViewReader, error)

	// blocks
	ApplyBlock(context.Context, *bc.Block, []*state.Output, map[bc.AssetID]*state.AssetState, *patricia.Tree) ([]*bc.Tx, error)
	LatestBlock(context.Context) (*bc.Block, error)
	NewViewForPrevouts(context.Context, []*bc.Tx) (state.ViewReader, error)
	StateTree(context.Context, uint64) (*patricia.Tree, error)
}

// FC provides a complete, minimal blockchain database. It
// delegates the underlying storage to other objects, and uses
// validation logic from package validation to decide what
// objects can be safely stored.
type FC struct {
	blockCallbacks []BlockCallback
	txCallbacks    []TxCallback
	trustedKeys    []*btcec.PublicKey
	height         struct {
		cond sync.Cond // protects n
		n    uint64
	}
	store Store
}

// New returns a new FC using store as the underlying storage.
//
// AddBlock will skip validation for any block signed by a key
// in trustedKeys. Typically, trustedKeys contains the public key
// for the local block-signer process; the presence of its
// signature indicates the block was already validated locally.
func New(ctx context.Context, store Store, trustedKeys []*btcec.PublicKey) (*FC, error) {
	fc := &FC{store: store, trustedKeys: trustedKeys}

	latestBlock, err := fc.LatestBlock(ctx)
	if err != nil && errors.Root(err) != ErrNoBlocks {
		return nil, errors.Wrap(err, "looking up latest block")
	}
	if latestBlock != nil {
		fc.height.n = latestBlock.Height
	}
	// Now fc.height.n may still be zero because of ErrNoBlocks.

	fc.height.cond.L = new(sync.Mutex)
	return fc, nil
}

func (fc *FC) AddBlockCallback(f BlockCallback) {
	fc.blockCallbacks = append(fc.blockCallbacks, f)
}

func (fc *FC) AddTxCallback(f TxCallback) {
	fc.txCallbacks = append(fc.txCallbacks, f)
}

func (fc *FC) LatestBlock(ctx context.Context) (*bc.Block, error) {
	b, err := fc.store.LatestBlock(ctx)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, ErrNoBlocks
	}
	return b, nil
}

func (fc *FC) WaitForBlock(ctx context.Context, height uint64) error {
	const slop = 3

	fc.height.cond.L.Lock()
	defer fc.height.cond.L.Unlock()

	if height > fc.height.n+slop {
		return ErrTheDistantFuture
	}

	for fc.height.n < height {
		fc.height.cond.Wait()
	}

	return nil
}
