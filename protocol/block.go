package protocol

import (
	"context"
	"fmt"
	"time"

	"chain/crypto/ed25519"
	"chain/errors"
	"chain/log"
	"chain/net/trace/span"
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
const saveSnapshotFrequency = 24 * time.Hour

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
// the current tx pool.  It returns the new block and a snapshot of what
// the state snapshot is if the block is applied. It has no side effects.
func (c *Chain) GenerateBlock(ctx context.Context, prev *bc.Block, snapshot *state.Snapshot, now time.Time) (b *bc.Block, result *state.Snapshot, err error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	timestampMS := bc.Millis(now)
	if timestampMS < prev.TimestampMS {
		return nil, nil, fmt.Errorf("timestamp %d is earlier than prevblock timestamp %d", timestampMS, prev.TimestampMS)
	}

	// Make a copy of the state that we can apply our changes to.
	result = state.Copy(snapshot)

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

	ctx = span.NewContextSuffix(ctx, "-validate-all")
	defer span.Finish(ctx)
	for _, tx := range txs {
		if len(b.Transactions) >= maxBlockTxs {
			break
		}

		if validation.ConfirmTx(result, tx, b.TimestampMS) == nil {
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
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	newState := state.Copy(prevState)
	err := validation.ValidateAndApplyBlock(ctx, newState, prev, block, c.validateTxCached)
	if err != nil {
		return nil, errors.Wrapf(ErrBadBlock, "validate block: %v", err)
	}
	return newState, nil
}

func (c *Chain) validateTxCached(tx *bc.Tx) error {
	// TODO(kr): consult a cache of prevalidated transactions.
	// It probably shouldn't use the pool, but instead keep
	// in memory a table of witness hashes (or similar).
	return validation.ValidateTx(tx)
}

// CommitBlock commits the block to the blockchain.
//
// This function:
//   * saves the block to the store.
//   * saves the state tree to the store (optionally).
//   * deletes any pending transactions that become conflicted
//     as a result of this block.
//   * executes all new-block callbacks.
//
// The block parameter must have already been validated before
// being committed.
func (c *Chain) CommitBlock(ctx context.Context, block *bc.Block, snapshot *state.Snapshot) error {
	err := c.commitBlock(ctx, block, snapshot)
	if err != nil {
		return errors.Wrap(err, "committing block")
	}

	_, err = c.rebuildPool(ctx, block, snapshot)
	return errors.Wrap(err, "rebuilding pool")
}

// commitBlock commits a block without rebuilding the pool.
func (c *Chain) commitBlock(ctx context.Context, block *bc.Block, snapshot *state.Snapshot) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

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

	for _, cb := range c.blockCallbacks {
		cb(ctx, block)
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
// block in preparation for signing it.  By definition it does not
// execute the sigscript.
func (c *Chain) ValidateBlockForSig(ctx context.Context, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

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
			// TODO(jackson): Forward request to leader process (who will have
			// the state snapshot in-memory).
			return ErrStaleState
		}
	}

	err := validation.ValidateBlockForSig(ctx, snapshot, prev, block, validation.ValidateTx)
	return errors.Wrap(err, "validation")
}

func (c *Chain) rebuildPool(ctx context.Context, block *bc.Block, snapshot *state.Snapshot) ([]*bc.Tx, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	txInBlock := make(map[bc.Hash]bool)
	for _, tx := range block.Transactions {
		txInBlock[tx.Hash] = true
	}

	var (
		deleteTxs   []*bc.Tx
		conflictTxs []*bc.Tx
	)

	txs, err := c.pool.Dump(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "dumping tx pool")
	}

	for _, tx := range txs {
		if block.TimestampMS < tx.MinTime {
			// This can't be confirmed yet because its mintime is too high.
			continue
		}

		txErr := validation.ConfirmTx(snapshot, tx, block.TimestampMS)
		if txErr == nil {
			validation.ApplyTx(snapshot, tx)
		} else {
			deleteTxs = append(deleteTxs, tx)
			if txInBlock[tx.Hash] {
				continue
			}

			// This should never happen in sandbox, unless a reservation expired
			// before the original tx was finalized.
			log.Messagef(ctx, "deleting conflict tx %v because %q", tx.Hash, txErr)
			conflictTxs = append(conflictTxs, tx)
		}
	}

	err = c.pool.Clean(ctx, deleteTxs)
	if err != nil {
		return nil, errors.Wrap(err, "removing conflicting txs")
	}
	return conflictTxs, nil
}

func NewInitialBlock(pubkeys []ed25519.PublicKey, nSigs int, timestamp time.Time) (*bc.Block, error) {
	script, err := vmutil.BlockMultiSigProgram(pubkeys, nSigs)
	if err != nil {
		return nil, err
	}
	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:          bc.NewBlockVersion,
			Height:           1,
			TimestampMS:      bc.Millis(timestamp),
			ConsensusProgram: script,
		},
	}
	return b, nil
}
