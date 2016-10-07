package protocol

import (
	"context"
	"fmt"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/validation"
)

// Recover performs crash recovery, restoring the blockchain
// to a complete state. It returns the latest confirmed block
// and the corresponding state snapshot.
//
// If the blockchain is empty (missing initial block), this function
// returns a nil block and a nil snapshot.
func (c *Chain) Recover(ctx context.Context) (*bc.Block, *state.Snapshot, error) {
	snapshot, snapshotHeight, err := c.store.LatestSnapshot(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting latest snapshot")
	}
	if snapshotHeight > 0 {
		snapshotBlock, err := c.store.GetBlock(ctx, snapshotHeight)
		if err != nil {
			return nil, nil, errors.Wrap(err, "getting snapshot block")
		}
		c.lastQueuedSnapshot = snapshotBlock.Time()
	}

	// The true height of the blockchain might be higher than the
	// height at which the state snapshot was taken. Replay all
	// existing blocks higher than the snapshot height.
	height, err := c.store.Height(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting blockchain height")
	}

	var b *bc.Block

	// Bring the snapshot up to date with the latest block
	for h := snapshotHeight + 1; h <= height; h++ {
		b, err = c.store.GetBlock(ctx, h)
		if err != nil {
			return nil, nil, errors.Wrap(err, "getting block")
		}
		err = validation.ApplyBlock(snapshot, b)
		if err != nil {
			return nil, nil, errors.Wrap(err, "applying block")
		}
		if b.AssetsMerkleRoot != snapshot.Tree.RootHash() {
			return nil, nil, fmt.Errorf("block %d has state root %s; snapshot has root %s",
				b.Height, b.AssetsMerkleRoot, snapshot.Tree.RootHash())
		}
	}
	if b != nil {
		// All blocks before the latest one have been fully processed
		// (saved in the db, callbacks invoked). The last one may have
		// been too, but make sure just in case. Also "finalize" the last
		// block (notifying other processes of the latest block height)
		// and maybe persist the snapshot.
		err = c.CommitBlock(ctx, b, snapshot)
		if err != nil {
			return nil, nil, errors.Wrap(err, "committing block")
		}
	}

	// For clarity, we always retrieve the latest block, even if it's redundant
	// with an earlier lookup above. It won't always be redundant, for ex, if
	// height == snapshotHeight.
	var tip *bc.Block
	if height > 0 {
		tip, err = c.store.GetBlock(ctx, height)
		if err != nil {
			return nil, nil, err
		}
		if tip.AssetsMerkleRoot != snapshot.Tree.RootHash() {
			return nil, nil, fmt.Errorf("block %d has state root %s; snapshot has root %s",
				tip.Height, tip.AssetsMerkleRoot, snapshot.Tree.RootHash())
		}
	}
	return tip, snapshot, nil
}
