/*

Package cos provides the core logic to tie together storage
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
outputs, and compose and sign the tx), call AddTx, and
send the transaction to a generator node. To wait for confirmation,
call WaitForBlock on successive block heights and inspect the
blockchain state (using GetTxs or GetBlock) until you find that
the transaction has been either confirmed or rejected.

To ingest a block, call AddBlock.

*/
package cos

import (
	"sync"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/crypto/ed25519"
	"chain/errors"
)

var (
	// ErrTheDistantFuture (https://youtu.be/2IPAOxrH7Ro) is returned when
	// waiting for a blockheight too far in excess of the tip of the
	// blockchain.
	ErrTheDistantFuture = errors.New("the block height is too damn high")

	// ErrNoBlocks is returned when LatestBlock is called and the store
	// contains no blocks.
	ErrNoBlocks = errors.New("no blocks in the store")

	// ErrBadStateHeight is returned from Store.StateTree when the
	// height parameter does not match the latest block height.
	ErrBadStateHeight = errors.New("requested block height does not match current state")
)

type BlockCallback func(ctx context.Context, block *bc.Block)
type TxCallback func(context.Context, *bc.Tx)

// Store provides storage for blockchain data: blocks, asset
// definition pointers and confirmed transactions.
//
// Note, this is different from a state tree. A state tree provides
// access to the state at a given point in time -- outputs and
// ADPs. An FC uses Store to load state trees from storage and
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

// FC provides a complete, minimal blockchain database. It
// delegates the underlying storage to other objects, and uses
// validation logic from package validation to decide what
// objects can be safely stored.
type FC struct {
	blockCallbacks []BlockCallback
	txCallbacks    []TxCallback
	trustedKeys    []ed25519.PublicKey
	height         struct {
		cond sync.Cond // protects n
		n    uint64
	}
	store Store
	pool  Pool
}

// NewFC returns a new FC using store as the underlying storage.
//
// AddBlock will skip validation for any block signed by a key
// in trustedKeys. Typically, trustedKeys contains the public key
// for the local block-signer process; the presence of its
// signature indicates the block was already validated locally.
func NewFC(ctx context.Context, store Store, pool Pool, trustedKeys []ed25519.PublicKey, heights <-chan uint64) (*FC, error) {
	fc := &FC{
		store:       store,
		pool:        pool,
		trustedKeys: trustedKeys,
	}
	fc.height.cond.L = new(sync.Mutex)

	var err error
	fc.height.n, err = store.Height(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "looking up blockchain height")
	}

	// Note that fc.height.n may still be zero here.
	if heights != nil {
		go func() {
			for h := range heights {
				fc.setHeight(h)
			}
		}()
	}

	return fc, nil
}

func (fc *FC) AddBlockCallback(f BlockCallback) {
	fc.blockCallbacks = append(fc.blockCallbacks, f)
}

func (fc *FC) AddTxCallback(f TxCallback) {
	fc.txCallbacks = append(fc.txCallbacks, f)
}

func (fc *FC) LatestBlock(ctx context.Context) (*bc.Block, error) {
	fc.height.cond.L.Lock()
	height := fc.height.n
	fc.height.cond.L.Unlock()

	b, err := fc.store.GetBlock(ctx, height)
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

// ConfirmedTxs looks up the provided hases in the confirmed blockchain.
func (fc *FC) ConfirmedTxs(ctx context.Context, hashes ...bc.Hash) (map[bc.Hash]*bc.Tx, error) {
	return fc.store.GetTxs(ctx, hashes...)
}

// PendingTxs looks up the provided hashes in the tx pool.
func (fc *FC) PendingTxs(ctx context.Context, hashes ...bc.Hash) (map[bc.Hash]*bc.Tx, error) {
	return fc.pool.GetTxs(ctx, hashes...)
}
