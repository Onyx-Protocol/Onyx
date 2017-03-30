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
func (c *Chain) GetBlock(ctx context.Context, height uint64) (*bc.Block, error) {
	return c.store.GetBlock(ctx, height)
}

// GenerateBlock generates a valid, but unsigned, candidate block from
// the current pending transaction pool. It returns the new block and
// a snapshot of what the state snapshot is if the block is applied.
//
// After generating the block, the pending transaction pool will be
// empty.
func (c *Chain) GenerateBlock(ctx context.Context, prev *bc.Block, snapshot *state.Snapshot, now time.Time, txs []*bc.Tx) (*bc.Block, *state.Snapshot, error) {
	timestampMS := bc.Millis(now)
	if timestampMS < prev.TimestampMS {
		return nil, nil, fmt.Errorf("timestamp %d is earlier than prevblock timestamp %d", timestampMS, prev.TimestampMS)
	}

	// Make a copy of the snapshot that we can apply our changes to.
	newSnapshot := state.Copy(c.state.snapshot)
	newSnapshot.PruneNonces(timestampMS)

	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:           bc.NewBlockVersion,
			Height:            prev.Height + 1,
			PreviousBlockHash: prev.Hash(),
			TimestampMS:       timestampMS,
			BlockCommitment: bc.BlockCommitment{
				ConsensusProgram: prev.ConsensusProgram,
			},
		},
	}

	var txEntries []*bc.TxEntries

	for _, tx := range txs {
		if len(b.Transactions) >= maxBlockTxs {
			break
		}

		// Filter out transactions that are not well-formed.
		err := c.ValidateTx(tx.TxEntries)
		if err != nil {
			// TODO(bobg): log this?
			continue
		}

		// Filter out double-spends etc.
		err = newSnapshot.ApplyTx(tx.TxEntries)
		if err != nil {
			// TODO(bobg): log this?
			continue
		}

		b.Transactions = append(b.Transactions, tx)
		txEntries = append(txEntries, tx.TxEntries)
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
func (c *Chain) ValidateBlock(block, prev *bc.Block) error {
	return validateBlock(block, prev, c.InitialBlockHash, c.ValidateTx, true)
}

func validateBlock(block, prev *bc.Block, initialBlockHash bc.Hash, validateTx func(*bc.TxEntries) error, runProg bool) error {
	err := validation.ValidateBlock(bc.MapBlock(block), bc.MapBlock(prev), initialBlockHash, validateTx, runProg)
	if err != nil {
		return errors.Sub(ErrBadBlock, err)
	}
	return nil
}

// ApplyValidBlock creates an updated snapshot without validating the
// block.
func (c *Chain) ApplyValidBlock(block *bc.Block) (*state.Snapshot, error) {
	newSnapshot := state.Copy(c.state.snapshot)
	err := newSnapshot.ApplyBlock(bc.MapBlock(block))
	if err != nil {
		return nil, err
	}
	if block.AssetsMerkleRoot != newSnapshot.Tree.RootHash() {
		return nil, ErrBadStateRoot
	}
	return newSnapshot, nil
}

// CommitBlock commits a block to the blockchain. The block
// must already have been applied with ApplyValidBlock or
// ApplyNewBlock, which will have produced the new snapshot that's
// required here.
//
// This function saves the block to the store and sometimes (not more
// often than saveSnapshotFrequency) saves the state tree to the
// store. New-block callbacks (via asynchronous block-processor pins)
// are triggered.
//
// TODO(bobg): rename to CommitAppliedBlock for clarity (deferred from https://github.com/chain/chain/pull/788)
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
		log.Printf(ctx, "snapshot storage is taking too long; last queued at %s",
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
// execute the consensus program.
func (c *Chain) ValidateBlockForSig(ctx context.Context, block *bc.Block) error {
	var prev *bc.Block

	if block.Height > 1 {
		var err error
		prev, err = c.GetBlock(ctx, block.Height-1)
		if err != nil {
			return errors.Wrap(err, "getting previous block")
		}
	}

	return validateBlock(block, prev, c.InitialBlockHash, c.ValidateTx, false)
}

func NewInitialBlock(pubkeys []ed25519.PublicKey, nSigs int, timestamp time.Time) (*bc.Block, error) {
	script, err := vmutil.BlockMultiSigProgram(pubkeys, nSigs)
	if err != nil {
		return nil, err
	}

	root, err := bc.MerkleRoot(nil) // calculate the zero value of the tx merkle root
	if err != nil {
		return nil, errors.Wrap(err, "calculating zero value of tx merkle root")
	}

	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:     bc.NewBlockVersion,
			Height:      1,
			TimestampMS: bc.Millis(timestamp),
			BlockCommitment: bc.BlockCommitment{
				TransactionsMerkleRoot: root,
				ConsensusProgram:       script,
			},
		},
	}
	return b, nil
}
