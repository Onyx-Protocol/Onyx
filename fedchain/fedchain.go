/*

Package fedchain provides the core logic to tie together storage
and validation for a blockchain. This comprises all behavior
that's common to every full node, as well as other functions that
need to operate on the blockchain state.

Here follows examples of typical full node types, but this is not
an exhaustive list.

Generator

An generator has two basic jobs: collecting transactions from
other nodes and putting them into blocks.

To collect pending transactions, call InsertPendingTx for each one.

To add a new block to the blockchain, call GenerateBlock, sign
the block (possibly collecting signatures from other parties),
and call InsertBlock.

Signer

An signer has one basic job: sign exactly one valid block at each height.

To ingest an unsigned block obtained from a generator node,
first validate against the current blockchain state
(TODO(kr): call ValidateBlockForSig), then sign the block, and
finally send the signature back to the generator node.

Manager

A manager's job is to select outputs for spending and to compose
transactions.

To publish a new transaction, prepare your transaction (select
outputs, and compose and sign the tx), call InsertPendingTx and
send the transaction to an generator node.
(TODO(kr): Then call WaitTx; when it returns (with no error),
the transaction has been confirmed.)

To ingest a block, call InsertBlock.

*/
package fedchain

import (
	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/btcec"

	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

var ErrTxRejected = errors.New("transaction rejected")

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
	ApplyTx(context.Context, *bc.Tx) error
	RemoveTxs(ctx context.Context, confirmed, conflicting []*bc.Tx) error
	PoolTxs(context.Context) ([]*bc.Tx, error)
	NewPoolViewForPrevouts(context.Context, []*bc.Tx) (state.ViewReader, error)

	// blocks
	ApplyBlock(context.Context, *bc.Block, map[bc.AssetID]*bc.AssetDefinitionPointer, []*state.Output) ([]*bc.Tx, error)
	LatestBlock(context.Context) (*bc.Block, error)
	NewViewForPrevouts(context.Context, []*bc.Tx) (state.ViewReader, error)
}

// FC provides a complete, minimal blockchain database. It
// delegates the underlying storage to other objects, and uses
// validation logic from package validation to decide what
// objects can be safely stored.
type FC struct {
	blockCallbacks []BlockCallback
	txCallbacks    []TxCallback
	trustedKeys    []*btcec.PublicKey
	store          Store
}

// New returns a new FC using store as the underlying storage.
//
// ApplyBlock will skip validation for any block signed by a key
// in trustedKeys. Typically, trustedKeys contains the public key
// for the local block-signer process; the presence of our own
// signature indicates the block was already validated locally.
func New(store Store, trustedKeys []*btcec.PublicKey) *FC {
	return &FC{store: store, trustedKeys: trustedKeys}
}

func (fc *FC) AddBlockCallback(f BlockCallback) {
	fc.blockCallbacks = append(fc.blockCallbacks, f)
}

func (fc *FC) AddTxCallback(f TxCallback) {
	fc.txCallbacks = append(fc.txCallbacks, f)
}
