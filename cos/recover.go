package cos

import (
	"context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/cos/validation"
	"chain/errors"
)

// Recover performs crash recovery, restoring the blockchain
// to a complete state. It returns the latest confirmed block
// and the corresponding state snapshot.
//
// If the blockchain is empty (missing genesis block), this function
// returns a nil block and a nil snapshot.
func (fc *FC) Recover(ctx context.Context) (*bc.Block, *state.Snapshot, error) {
	snapshot, snapshotHeight, err := fc.store.LatestSnapshot(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting latest snapshot")
	}

	// The true height of the blockchain might be higher than the
	// height at which the state snapshot was taken. Replay all
	// existing blocks higher than the snapshot height.
	height, err := fc.store.Height(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "getting blockchain height")
	}
	for h := snapshotHeight + 1; h <= height; h++ {
		b, err := fc.store.GetBlock(ctx, h)
		if err != nil {
			return nil, nil, errors.Wrap(err, "getting block")
		}
		err = validation.ApplyBlock(snapshot, b)
		if err != nil {
			return nil, nil, errors.Wrap(err, "applying block")
		}

		// Commit the block again in case we crashed halfway through
		// and the block isn't fully committed.
		// TODO(jackson): Calling CommitBlock() is overboard and performs
		// a lot of redundant work. Only do what is necessary.
		err = fc.CommitBlock(ctx, b, snapshot)
		if err != nil {
			return nil, nil, errors.Wrap(err, "committing block")
		}
	}

	// For clarity, we always retrieve the latest block, even if it's redundant
	// with an earlier lookup above. It won't always be redundant, for ex, if
	// height == snapshotHeight.
	var tip *bc.Block
	if height > 0 {
		tip, err = fc.store.GetBlock(ctx, height)
		if err != nil {
			return nil, nil, err
		}
	}
	return tip, snapshot, nil
}
