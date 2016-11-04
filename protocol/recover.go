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
// returns a nil block and an empty snapshot.
func (c *Chain) Recover(ctx context.Context) (*bc.Block, *state.Snapshot, error) {
	snapshot, snapshotHeight, err := c.store.LatestSnapshot(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting latest snapshot")
	}
	var b *bc.Block
	if snapshotHeight > 0 {
		b, err = c.store.GetBlock(ctx, snapshotHeight)
		if err != nil {
			return nil, nil, errors.Wrap(err, "getting snapshot block")
		}
		c.lastQueuedSnapshot = b.Time()
	}
	if snapshot == nil {
		snapshot = state.Empty()
	}

	// The true height of the blockchain might be higher than the
	// height at which the state snapshot was taken. Replay all
	// existing blocks higher than the snapshot height.
	height, err := c.store.Height(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting blockchain height")
	}

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

	close(c.ready)

	return b, snapshot, nil
}
