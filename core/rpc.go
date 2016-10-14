package core

import (
	"context"

	"chain/encoding/json"
	"chain/errors"
)

// getBlockRPC returns the block at the requested height.
// If successful, it always returns at least one block,
// waiting if necessary until one is created.
// It is an error to request blocks very far in the future.
func (h *Handler) getBlockRPC(ctx context.Context, height uint64) (json.HexBytes, error) {
	err := h.Chain.WaitForBlockSoon(ctx, height)
	if err != nil {
		return nil, errors.Wrapf(err, "waiting for block at height %d", height)
	}

	rawBlock, err := h.Store.GetRawBlock(ctx, height)
	if err != nil {
		return nil, err
	}

	return rawBlock, nil
}

// Data is a []byte because it's being funneled from the
// generator's db to the recipient node's db, and this is
// the smallest serialization format.
type snapshotResp struct {
	Data   []byte `json:"data"`
	Height uint64 `json:"height"`
}

// getSnapshotRPC returns the latest snapshot data.
// The generator should run this to bootstrap new cores.
// Non-generators can call this endpoint to get raw data
// that they can use to populate their own snapshot table.
func (h *Handler) getSnapshotRPC(ctx context.Context) (resp snapshotResp, err error) {
	resp.Data, resp.Height, err = h.Store.LatestFullSnapshot(ctx)
	return resp, err
}
