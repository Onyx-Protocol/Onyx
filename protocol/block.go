package protocol

import (
	"context"
	"fmt"
	"time"

	"chain/crypto/ed25519"
	"chain/errors"
	"chain/log"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/validation"
	"chain/protocol/vmutil"
)

// maxBlockTxs limits the number of transactions
// included in each block.
const maxBlockTxs = 10000

// saveSnapshotFrequency stores how often to save a state
// snapshot to the Store.
const saveSnapshotFrequency = time.Hour

// ErrBadBlock is returned when a block is invalid.
var ErrBadBlock = errors.New("invalid block")

// ErrStaleState is returned when the Chain does not have a current
// blockchain state.
var ErrStaleState = errors.New("stale blockchain state")

// GetBlock returns the block at the given height, if there is one,
// otherwise it returns an error.
func (c *Chain) GetBlock(ctx context.Context, height uint64) (*bc.Block, error) {
	return c.store.GetBlock(ctx, height)
}

// GenerateBlock generates a valid, but unsigned, candidate block from
// the current pending transaction pool. It returns the new block and
// a snapshot of what the state snapshot is if the block is applied.
//
// After generating the block, the pending transaction pool will be
// empty.
func (c *Chain) GenerateBlock(ctx context.Context, prev *bc.Block, snapshot *state.Snapshot, now time.Time) (b *bc.Block, result *state.Snapshot, err error) {
	timestampMS := bc.Millis(now)
	if timestampMS < prev.TimestampMS {
		return nil, nil, fmt.Errorf("timestamp %d is earlier than prevblock timestamp %d", timestampMS, prev.TimestampMS)
	}

	// Make a copy of the state that we can apply our changes to.
	result = state.Copy(snapshot)
	result.PruneIssuances(timestampMS)

	txs, err := c.pool.Dump(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "get pool TXs")
	}

	b = &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:           bc.NewBlockVersion,
			Height:            prev.Height + 1,
			PreviousBlockHash: prev.Hash(),
			TimestampMS:       timestampMS,
			ConsensusProgram:  prev.ConsensusProgram,
		},
	}

	for _, tx := range txs {
		if len(b.Transactions) >= maxBlockTxs {
			break
		}

		if validation.ConfirmTx(result, c.InitialBlockHash, b, tx) == nil {
			validation.ApplyTx(result, tx)
			b.Transactions = append(b.Transactions, tx)
		}
	}
	b.TransactionsMerkleRoot = validation.CalcMerkleRoot(b.Transactions)
	b.AssetsMerkleRoot = result.Tree.RootHash()
	return b, result, nil
}

// ValidateBlock performs validation on an incoming block, in advance
// of committing the block. ValidateBlock returns the state after
// the block has been applied.
func (c *Chain) ValidateBlock(ctx context.Context, prevState *state.Snapshot, prev, block *bc.Block) (*state.Snapshot, error) {
	newState := state.Copy(prevState)
	err := validation.ValidateBlockForAccept(ctx, newState, c.InitialBlockHash, prev, block, c.ValidateTxCached)
	if err != nil {
		return nil, errors.Wrapf(ErrBadBlock, "validate block: %v", err)
	}
	// TODO(kr): consider calling CommitBlock here and
	// renaming this function to AcceptBlock.
	// See $CHAIN/protocol/doc/spec/validation.md#accept-block
	// and the comment in validation/block.go:/ValidateBlock.
	return newState, nil
}

// CommitBlock commits the block to the blockchain.
//
// This function:
//   * saves the block to the store.
//   * saves the state tree to the store (optionally).
//   * executes all new-block callbacks.
//
// The block parameter must have already been validated before
// being committed.
func (c *Chain) CommitBlock(ctx context.Context, block *bc.Block, snapshot *state.Snapshot) error {
	// SaveBlock is the linearization point. Once the block is committed
	// to persistent storage, the block has been applied and everything
	// else can be derived from that block.
	err := c.store.SaveBlock(ctx, block)
	if err != nil {
		return errors.Wrap(err, "storing block")
	}
	if block.Time().After(c.lastQueuedSnapshot.Add(saveSnapshotFrequency)) {
		c.queueSnapshot(ctx, block.Height, block.Time(), snapshot)
	}

	err = c.store.FinalizeBlock(ctx, block.Height)
	if err != nil {
		return errors.Wrap(err, "finalizing block")
	}

	// c.setState will update the local blockchain state and height.
	// When c.store is a txdb.Store, and c has been initialized with a
	// channel from txdb.ListenBlocks, then the above call to
	// c.store.FinalizeBlock will have done a postgresql NOTIFY and
	// that will wake up the goroutine in NewChain, which also calls
	// setHeight.  But duplicate calls with the same blockheight are
	// harmless; and the following call is required in the cases where
	// it's not redundant.
	c.setState(block, snapshot)
	return nil
}

func (c *Chain) queueSnapshot(ctx context.Context, height uint64, timestamp time.Time, s *state.Snapshot) {
	// Non-blockingly queue the snapshot for storage.
	ps := pendingSnapshot{height: height, snapshot: s}
	select {
	case c.pendingSnapshots <- ps:
		c.lastQueuedSnapshot = timestamp
	default:
		// Skip it; saving snapshots is taking longer than the snapshotting period.
		log.Messagef(ctx, "snapshot storage is taking too long; last queued at %s",
			c.lastQueuedSnapshot)
	}
}

func (c *Chain) setHeight(h uint64) {
	// We call setHeight from two places independently:
	// CommitBlock and the Postgres LISTEN goroutine.
	// This means we can get here twice for each block,
	// and any of them might be arbitrarily delayed,
	// which means h might be from the past.
	// Detect and discard these duplicate calls.

	c.state.cond.L.Lock()
	defer c.state.cond.L.Unlock()

	if h <= c.state.height {
		return
	}
	c.state.height = h
	c.state.cond.Broadcast()
}

// ValidateBlockForSig performs validation on an incoming _unsigned_
// block in preparation for signing it. By definition it does not
// execute the sigscript.
func (c *Chain) ValidateBlockForSig(ctx context.Context, block *bc.Block) error {
	var (
		prev     *bc.Block
		snapshot = state.Empty()
	)

	if block.Height > 1 {
		var err error
		prev, err = c.store.GetBlock(ctx, block.Height-1)
		if err != nil {
			return errors.Wrap(err, "getting previous block")
		}

		prev, snapshot = c.State()
		if prev == nil || prev.Height != block.Height-1 {
			return ErrStaleState
		}
	}

	// TODO(kr): cache the applied snapshot, and maybe
	// we can skip re-applying it later
	snapshot = state.Copy(snapshot)
	err := validation.ValidateBlock(ctx, snapshot, c.InitialBlockHash, prev, block, validation.CheckTxWellFormed)
	return errors.Wrap(err, "validation")
}

func NewInitialBlock(pubkeys []ed25519.PublicKey, nSigs int, timestamp time.Time) (*bc.Block, error) {
	script, err := vmutil.BlockMultiSigProgram(pubkeys, nSigs)
	if err != nil {
		return nil, err
	}
	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:                bc.NewBlockVersion,
			Height:                 1,
			TimestampMS:            bc.Millis(timestamp),
			ConsensusProgram:       script,
			TransactionsMerkleRoot: validation.CalcMerkleRoot([]*bc.Tx{}), // calculate the zero value of the tx merkle root
		},
	}
	return b, nil
}
