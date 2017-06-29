package protocol

import (
	"context"
	"fmt"
	"time"

	"chain/crypto/ed25519"
	"chain/errors"
	"chain/log"
	"chain/protocol/bc"
	"chain/protocol/bc/bcvm"
	"chain/protocol/state"
	"chain/protocol/validation"
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
func (c *Chain) GetBlock(ctx context.Context, height uint64) (*bcvm.Block, error) {
	return c.store.GetBlock(ctx, height)
}

// GenerateBlock generates a valid, but unsigned, candidate block from
// the current pending transaction pool. It returns the new block and
// a snapshot of what the state snapshot is if the block is applied.
//
// After generating the block, the pending transaction pool will be
// empty.
func (c *Chain) GenerateBlock(ctx context.Context, prev *bcvm.Block, snapshot *state.Snapshot, now time.Time, txs [][]byte) (*bcvm.Block, *state.Snapshot, error) {
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

	b := &bcvm.Block{
		BlockHeader: bcvm.BlockHeader{
			Version:           1,
			Height:            prev.Height + 1,
			PreviousBlockHash: prev.Hash(),
			TimestampMS:       timestampMS,
			BlockCommitment: bcvm.BlockCommitment{
				ConsensusPubkeys: prev.ConsensusPubkeys,
				ConsensusQuorum:  prev.ConsensusQuorum,
			},
		},
	}

	var ids []bc.Hash

	for _, tx := range txs {
		if len(b.Transactions) >= maxBlockTxs {
			break
		}

		// Filter out transactions that are not well-formed.
		err := c.ValidateTx(tx)
		if err != nil {
			// TODO(bobg): log this?
			continue
		}

		deserialized, err := bcvm.NewTx(tx)
		if err != nil {
			continue
		}

		timestampsValid := true
		for _, tc := range deserialized.TimeConstraints {
			if (tc.Type == "min" && b.TimestampMS < uint64(tc.Time)) || (tc.Type == "max" && b.TimestampMS > uint64(tc.Time)) {
				timestampsValid = false
				break
			}
		}
		if !timestampsValid {
			continue
		}

		// Filter out double-spends etc.
		err = newSnapshot.ApplyTx(deserialized)
		if err != nil {
			// TODO(bobg): log this?
			continue
		}

		b.Transactions = append(b.Transactions, tx)
		ids = append(ids, deserialized.ID)
	}

	var err error

	b.TransactionsMerkleRoot, err = bcvm.MerkleRoot(ids)
	if err != nil {
		return nil, nil, errors.Wrap(err, "calculating tx merkle root")
	}

	b.AssetsMerkleRoot = newSnapshot.Tree.RootHash()

	return b, newSnapshot, nil
}

// ValidateBlock validates an incoming block in advance of applying it
// to a snapshot (with ApplyValidBlock) and committing it to the
// blockchain (with CommitAppliedBlock).
func (c *Chain) ValidateBlock(block, prev *bcvm.Block) error {
	err := validation.ValidateBlock(block, prev)
	if err != nil {
		return errors.Sub(ErrBadBlock, err)
	}
	if block.Height > 1 {
		err = validation.ValidateBlockSig(block, prev.ConsensusPubkeys, prev.ConsensusQuorum)
	}
	return errors.Sub(ErrBadBlock, err)
}

// ApplyValidBlock creates an updated snapshot without validating the
// block.
func (c *Chain) ApplyValidBlock(block *bcvm.Block) (*state.Snapshot, error) {
	newSnapshot := state.Copy(c.state.snapshot)
	err := newSnapshot.ApplyBlock(block)
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
func (c *Chain) CommitAppliedBlock(ctx context.Context, block *bcvm.Block, snapshot *state.Snapshot) error {
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
func (c *Chain) ValidateBlockForSig(ctx context.Context, block *bcvm.Block) error {
	var prev *bcvm.Block

	if block.Height > 1 {
		var err error
		prev, err = c.GetBlock(ctx, block.Height-1)
		if err != nil {
			return errors.Wrap(err, "getting previous block")
		}
	}

	err := validation.ValidateBlock(block, prev)
	return errors.Sub(ErrBadBlock, err)
}

func NewInitialBlock(pubkeys []ed25519.PublicKey, nSigs int, timestamp time.Time) (*bcvm.Block, error) {
	// TODO(kr): move this into a lower-level package (e.g. chain/protocol/bc)
	// so that other packages (e.g. chain/protocol/validation) unit tests can
	// call this function.

	root, err := bc.MerkleRoot(nil) // calculate the zero value of the tx merkle root
	if err != nil {
		return nil, errors.Wrap(err, "calculating zero value of tx merkle root")
	}

	var keybytes [][]byte
	for _, k := range pubkeys {
		keybytes = append(keybytes, []byte(k))
	}

	b := &bcvm.Block{
		BlockHeader: bcvm.BlockHeader{
			Version:     1,
			Height:      1,
			TimestampMS: bc.Millis(timestamp),
			BlockCommitment: bcvm.BlockCommitment{
				TransactionsMerkleRoot: root,
				ConsensusPubkeys:       keybytes,
				ConsensusQuorum:        uint32(nSigs),
			},
		},
	}
	return b, nil
}
