package protocol

import (
	"context"
	"fmt"
	"time"

	"chain/crypto/ed25519"
	"chain/errors"
	"chain/log"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/state"
	"chain/protocol/validation"
	"chain/protocol/vm/vmutil"
)

// maxBlockTxs limits the number of transactions
// included in each block.
const maxBlockTxs = 10000

// saveSnapshotFrequency stores how often to save a state
// snapshot to the Store.
const saveSnapshotFrequency = time.Hour

var (
	// ErrBadBlock is returned when a block is invalid.
	ErrBadBlock = errors.New("invalid block")

	// ErrStaleState is returned when the Chain does not have a current
	// blockchain state.
	ErrStaleState = errors.New("stale blockchain state")

	// ErrBadStateRoot is returned when the computed assets merkle root
	// disagrees with the one declared in a block header.
	ErrBadStateRoot = errors.New("invalid state merkle root")
)

// GetBlock returns the block at the given height, if there is one,
// otherwise it returns an error.
func (c *Chain) GetBlock(ctx context.Context, height uint64) (*legacy.Block, error) {
	return c.store.GetBlock(ctx, height)
}

// GenerateBlock generates a valid, but unsigned, candidate block from
// the current pending transaction pool. It returns the new block and
// a snapshot of what the state snapshot is if the block is applied.
//
// After generating the block, the pending transaction pool will be
// empty.
func (c *Chain) GenerateBlock(ctx context.Context, prev *legacy.Block, snapshot *state.Snapshot, now time.Time, txs []*legacy.Tx) (*legacy.Block, *state.Snapshot, error) {
	// TODO(kr): move this into a lower-level package (e.g. chain/protocol/bc)
	// so that other packages (e.g. chain/protocol/validation) unit tests can
	// call this function.

	timestampMS := bc.Millis(now)
	if timestampMS < prev.TimestampMS {
		return nil, nil, fmt.Errorf("timestamp %d is earlier than prevblock timestamp %d", timestampMS, prev.TimestampMS)
	}

	// Make a copy of the snapshot that we can apply our changes to.
	newSnapshot := state.Copy(c.state.snapshot)
	newSnapshot.PruneNonces(timestampMS)

	b := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:           1,
			Height:            prev.Height + 1,
			PreviousBlockHash: prev.Hash(),
			TimestampMS:       timestampMS,
			BlockCommitment: legacy.BlockCommitment{
				ConsensusProgram: prev.ConsensusProgram,
			},
		},
	}

	var txEntries []*bc.Tx

	for _, tx := range txs {
		if len(b.Transactions) >= maxBlockTxs {
			break
		}

		// Filter out transactions that are not well-formed.
		err := c.ValidateTx(tx.Tx)
		if err != nil {
			// TODO(bobg): log this?
			continue
		}

		// Filter out transactions that are not yet valid, or no longer
		// valid, per the block's timestamp.
		if tx.Tx.MinTimeMs > 0 && tx.Tx.MinTimeMs > b.TimestampMS {
			// TODO(bobg): log this?
			continue
		}
		if tx.Tx.MaxTimeMs > 0 && tx.Tx.MaxTimeMs < b.TimestampMS {
			// TODO(bobg): log this?
			continue
		}

		// Filter out double-spends etc.
		err = newSnapshot.ApplyTx(tx.Tx)
		if err != nil {
			// TODO(bobg): log this?
			continue
		}

		b.Transactions = append(b.Transactions, tx)
		txEntries = append(txEntries, tx.Tx)
	}

	var err error

	b.TransactionsMerkleRoot, err = bc.MerkleRoot(txEntries)
	if err != nil {
		return nil, nil, errors.Wrap(err, "calculating tx merkle root")
	}

	b.AssetsMerkleRoot = newSnapshot.Tree.RootHash()

	return b, newSnapshot, nil
}

// ValidateBlock validates an incoming block in advance of applying it
// to a snapshot (with ApplyValidBlock) and committing it to the
// blockchain (with CommitAppliedBlock).
func (c *Chain) ValidateBlock(block, prev *legacy.Block) error {
	blockEnts := legacy.MapBlock(block)
	prevEnts := legacy.MapBlock(prev)
	err := validation.ValidateBlock(blockEnts, prevEnts, c.InitialBlockHash, c.ValidateTx)
	if err != nil {
		return errors.Sub(ErrBadBlock, err)
	}
	if block.Height > 1 {
		err = validation.ValidateBlockSig(blockEnts, prevEnts.NextConsensusProgram)
	}
	return errors.Sub(ErrBadBlock, err)
}

// ApplyValidBlock creates an updated snapshot without validating the
// block.
func (c *Chain) ApplyValidBlock(block *legacy.Block) (*state.Snapshot, error) {
	_, oldSnapshot := c.State()
	newSnapshot := state.Copy(oldSnapshot)
	err := newSnapshot.ApplyBlock(legacy.MapBlock(block))
	if err != nil {
		return nil, err
	}
	if block.AssetsMerkleRoot != newSnapshot.Tree.RootHash() {
		return nil, ErrBadStateRoot
	}
	return newSnapshot, nil
}

// CommitAppliedBlock commits a block to the blockchain. The block
// must already have been applied with ApplyValidBlock or
// ApplyNewBlock, which will have produced the new snapshot that's
// required here.
//
// This function saves the block to the store and sometimes (not more
// often than saveSnapshotFrequency) saves the state tree to the
// store. New-block callbacks (via asynchronous block-processor pins)
// are triggered.
func (c *Chain) CommitAppliedBlock(ctx context.Context, block *legacy.Block, snapshot *state.Snapshot) error {
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

	// c.setState will update the local blockchain state and height.
	// When c.store is a txdb.Store, and c has been initialized with a
	// channel from txdb.ListenBlocks, then the below call to
	// c.store.FinalizeBlock will have done a postgresql NOTIFY and
	// that will wake up the applyCommittedState goroutine, which also
	// calls setState. But duplicate calls with the same block/height are
	// harmless; and the following call is required in the cases where
	// it's not redundant. We call setState before FinalizeBlock to
	// avoid applying the block twice within this process.
	c.setState(block, snapshot)

	err = c.store.FinalizeBlock(ctx, block.Height)
	if err != nil {
		return errors.Wrap(err, "finalizing block")
	}

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
		log.Printf(ctx, "snapshot storage is taking too long; last queued at %s",
			c.lastQueuedSnapshot)
	}
}

// applyCommittedState reads state as it's committed and applies it to c.
// It runs as a goroutine launched from NewChain. Only applyCommittedState
// should update c.state.
func (c *Chain) applyCommittedState(ctx context.Context, committedHeights <-chan uint64) {
	committedHeight := c.Height()
	for {
		appliedHeight := c.Height()

		// If we know there's a committed but unapplied block, apply it.
		if committedHeight > appliedHeight {
			b, err := c.store.GetBlock(ctx, appliedHeight+1)
			if err != nil {
				// TODO(jackson): should this error be exposed via monitoring endpoints?
				log.Error(ctx, err, "at", "retrieving committed block", "height", appliedHeight+1)
				continue
			}
			s, err := c.ApplyValidBlock(b)
			if err != nil {
				// TODO(jackson): should this error be exposed via monitoring endpoints?
				log.Error(ctx, err, "at", "applying committed blocks", "height", b.Height)
				continue
			}
			c.setState(b, s)

			// NOTE(jackson): Once we're storing blockchain snapshots in localdb,
			// we'll need to queue snapshots for storage here.
			continue
		}

		// If we've reached here, we don't know of any committed blocks
		// that haven't been applied to c's State. Wait for the next
		// notification of a new committed block.
		select {
		case h := <-committedHeights:
			if h > committedHeight {
				committedHeight = h
			}
		case <-ctx.Done():
			return
		}
	}
}

// ValidateBlockForSig performs validation on an incoming _unsigned_
// block in preparation for signing it. By definition it does not
// execute the consensus program.
func (c *Chain) ValidateBlockForSig(ctx context.Context, block *legacy.Block) error {
	var prev *legacy.Block

	if block.Height > 1 {
		var err error
		prev, err = c.GetBlock(ctx, block.Height-1)
		if err != nil {
			return errors.Wrap(err, "getting previous block")
		}
	}

	err := validation.ValidateBlock(legacy.MapBlock(block), legacy.MapBlock(prev), c.InitialBlockHash, c.ValidateTx)
	return errors.Sub(ErrBadBlock, err)
}

func NewInitialBlock(pubkeys []ed25519.PublicKey, nSigs int, timestamp time.Time) (*legacy.Block, error) {
	// TODO(kr): move this into a lower-level package (e.g. chain/protocol/bc)
	// so that other packages (e.g. chain/protocol/validation) unit tests can
	// call this function.

	script, err := vmutil.BlockMultiSigProgram(pubkeys, nSigs)
	if err != nil {
		return nil, err
	}

	root, err := bc.MerkleRoot(nil) // calculate the zero value of the tx merkle root
	if err != nil {
		return nil, errors.Wrap(err, "calculating zero value of tx merkle root")
	}

	b := &legacy.Block{
		BlockHeader: legacy.BlockHeader{
			Version:     1,
			Height:      1,
			TimestampMS: bc.Millis(timestamp),
			BlockCommitment: legacy.BlockCommitment{
				TransactionsMerkleRoot: root,
				ConsensusProgram:       script,
			},
		},
	}
	return b, nil
}
