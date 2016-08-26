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

// saveSnapshotFrequency stores how often (in blocks) to save a state
// snapshot to the Store.
const saveSnapshotFrequency = 10

// ErrBadBlock is returned when a block is invalid.
var ErrBadBlock = errors.New("invalid block")

// GenerateBlock generates a valid, but unsigned, candidate block from
// the current tx pool.  It returns the new block and has no side effects.
func (c *Chain) GenerateBlock(ctx context.Context, prev *bc.Block, snapshot *state.Snapshot, now time.Time) (b *bc.Block, err error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	timestampMS := bc.Millis(now)
	if timestampMS < prev.TimestampMS {
		return nil, fmt.Errorf("timestamp %d is earlier than prevblock timestamp %d", timestampMS, prev.TimestampMS)
	}

	// Make a copy of the state that we can apply our changes to.
	snapshot = state.Copy(snapshot)

	txs, err := c.pool.Dump(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get pool TXs")
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

		if validation.ConfirmTx(snapshot, tx, b.TimestampMS) == nil {
			validation.ApplyTx(snapshot, tx)
			b.Transactions = append(b.Transactions, tx)
		}
	}
	b.TransactionsMerkleRoot = validation.CalcMerkleRoot(b.Transactions)
	b.AssetsMerkleRoot = snapshot.Tree.RootHash()
	return b, nil
}

// ValidateBlock performs validation on an incoming block, in advance
// of committing the block. ValidateBlock returns the state after
// the block has been applied.
func (c *Chain) ValidateBlock(ctx context.Context, prevState *state.Snapshot, prev, block *bc.Block) (*state.Snapshot, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	err := validation.ValidateBlockHeader(prev, block)
	if err != nil {
		return nil, errors.Wrap(err, "validating block header")
	}

	newState := state.Copy(prevState)
	if isSignedByTrustedHost(block, c.trustedKeys) {
		err = validation.ApplyBlock(newState, block)
	} else {
		err = validation.ValidateAndApplyBlock(ctx, newState, prev, block)
	}
	if err != nil {
		return nil, errors.Wrapf(ErrBadBlock, "validate block: %v", err)
	}
	return newState, nil
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

	// Save a state snapshot periodically.
	if block.Height%saveSnapshotFrequency == 0 {
		// TODO(jackson): Save the snapshot asnychronously, but ensure
		// that we never fall behind if saving a snapshot takes longer
		// than the snapshotting period.
		err = c.store.SaveSnapshot(ctx, block.Height, snapshot)
		if err != nil {
			return errors.Wrap(err, "storing state snapshot")
		}
	}

	for _, cb := range c.blockCallbacks {
		cb(ctx, block)
	}

	err = c.store.FinalizeBlock(ctx, block.Height)
	if err != nil {
		return errors.Wrap(err, "finalizing block")
	}

	// When c.store is a txdb.Store, and c has been initialized with a
	// channel from txdb.ListenBlocks, then the above call to
	// c.store.FinalizeBlock will have done a postgresql NOTIFY and
	// that will wake up the goroutine in NewChain, which also calls
	// setHeight.  But duplicate calls with the same blockheight are
	// harmless; and the following call is required in the cases where
	// it's not redundant.
	c.setHeight(block.Height)
	return nil
}

func (c *Chain) setHeight(h uint64) {
	// We call setHeight from two places independently:
	// CommitBlock and the Postgres LISTEN goroutine.
	// This means we can get here twice for each block,
	// and any of them might be arbitrarily delayed,
	// which means h might be from the past.
	// Detect and discard these duplicate calls.

	c.height.cond.L.Lock()
	defer c.height.cond.L.Unlock()

	if h <= c.height.n {
		return
	}
	c.height.n = h
	c.height.cond.Broadcast()
}

func (c *Chain) currentState(ctx context.Context, expectedHeight uint64) (*state.Snapshot, error) {
	// TODO(jackson): Store the state tree on Chain.
	snapshot, height, err := c.store.LatestSnapshot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "loading state snapshot")
	}
	if height != expectedHeight {
		return nil, errors.New("missing state snapshot for block")
	}
	return state.Copy(snapshot), nil
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

		// TODO(jackson): Forward request to leader process (who will have
		// the state snapshot in-memory) and delete `c.currentState`.
		// Because we don't save the snapshot on every block, this isn't
		// guaranteed to be correct!
		snapshot, err = c.currentState(ctx, prev.Height)
		if err != nil {
			return err
		}
	}

	err := validation.ValidateBlockForSig(ctx, snapshot, prev, block)
	return errors.Wrap(err, "validation")
}

func isSignedByTrustedHost(block *bc.Block, trustedKeys []ed25519.PublicKey) bool {
	hash := block.HashForSig()
	for _, sig := range block.Witness {
		if len(sig) == 0 {
			continue
		}
		for _, pubk := range trustedKeys {
			if ed25519.Verify(pubk, hash[:], sig) {
				return true
			}
		}
	}

	return false
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

// ComputeBlockSignature signs a block with the given key.  It does
// not validate the block.
func ComputeBlockSignature(b *bc.Block, key ed25519.PrivateKey) []byte {
	hash := b.HashForSig()
	return ed25519.Sign(key, hash[:])
}

// AddSignaturesToBlock adds signatures to a block, replacing the
// block's SignatureScript.  The signatures must be in the correct
// order, to wit: matching the order of pubkeys in the previous
// block's output script.
func AddSignaturesToBlock(b *bc.Block, signatures [][]byte) {
	b.Witness = append([][]byte{}, signatures...)
}

// GenerateBlockScript generates a predicate script
// requiring nSigs signatures from the given keys.
func GenerateBlockScript(keys []ed25519.PublicKey, nSigs int) ([]byte, error) {
	return vmutil.BlockMultiSigScript(keys, nSigs)
}

func NewGenesisBlock(pubkeys []ed25519.PublicKey, nSigs int, timestamp time.Time) (*bc.Block, error) {
	script, err := GenerateBlockScript(pubkeys, nSigs)
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
