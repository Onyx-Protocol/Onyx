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

// ValidateBlock validates an incoming block in advance of committing
// it to the blockchain (with CommitBlock).
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

// CommitAppliedBlock takes a block, commits it to persistent storage and
// sets c's state. Unlike CommitBlock, it accepts an already applied
// snapshot. CommitAppliedBlock is idempotent.
func (c *Chain) CommitAppliedBlock(ctx context.Context, block *legacy.Block, snapshot *state.Snapshot) error {
	err := c.store.SaveBlock(ctx, block)
	if err != nil {
		return errors.Wrap(err, "storing block")
	}
	curBlock, _ := c.State()

	// CommitAppliedBlock needs to be idempotent. If block's height is less than or
	// equal to c's current block, then it was already applied. Because
	// SaveBlock didn't error with a conflict, we know it's not a different
	// block at the same height.
	if curBlock != nil && block.Height <= curBlock.Height {
		return nil
	}
	return c.finalizeCommitBlock(ctx, block, snapshot)
}

// CommitBlock takes a block, commits it to persistent storage and applies
// it to c. CommitBlock is idempotent. A duplicate call with a previously
// committed block will succeed.
func (c *Chain) CommitBlock(ctx context.Context, block *legacy.Block) error {
	err := c.store.SaveBlock(ctx, block)
	if err != nil {
		return errors.Wrap(err, "storing block")
	}
	curBlock, curSnapshot := c.State()

	// CommitBlock needs to be idempotent. If block's height is less than or
	// equal to c's current block, then it was already applied. Because
	// SaveBlock didn't error with a conflict, we know it's not a different
	// block at the same height.
	if curBlock != nil && block.Height <= curBlock.Height {
		return nil
	}

	snapshot := state.Copy(curSnapshot)
	err = snapshot.ApplyBlock(legacy.MapBlock(block))
	if err != nil {
		return err
	}
	if block.AssetsMerkleRoot != snapshot.Tree.RootHash() {
		return ErrBadStateRoot
	}
	return c.finalizeCommitBlock(ctx, block, snapshot)
}

func (c *Chain) finalizeCommitBlock(ctx context.Context, block *legacy.Block, snapshot *state.Snapshot) error {
	// Save the blockchain state tree snapshot to persistent storage
	// if we haven't done it recently.
	if block.Time().After(c.lastQueuedSnapshot.Add(saveSnapshotFrequency)) {
		c.queueSnapshot(ctx, block.Height, block.Time(), snapshot)
	}

	// setState will update c's current block and snapshot, or no-op
	// if another goroutine has already updated the state.
	c.setState(block, snapshot)

	// The below FinalizeBlock will notify other cored processes that
	// the a new block has been committed. It may result in a duplicate
	// attempt to update c's height but setState and setHeight safely
	// ignore duplicate heights.
	err := c.store.FinalizeBlock(ctx, block.Height)
	return errors.Wrap(err, "finalizing block")
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
