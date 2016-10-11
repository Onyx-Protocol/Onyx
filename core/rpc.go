package core

import (
	"context"

	"chain/core/txdb"
	"chain/encoding/json"
	"chain/errors"
)

// getBlocksRPC returns contiguous blocks
// with heights larger than afterHeight,
// in block-height order.
// If successful, it always returns at least one block,
// waiting if necessary until one is created.
// It is not guaranteed to return all available blocks.
// It is an error to request blocks very far in the future.
func (h *Handler) getBlocksRPC(ctx context.Context, afterHeight uint64) ([]json.HexBytes, error) {
	err := h.Chain.WaitForBlockSoon(ctx, afterHeight+1)
	if err != nil {
		return nil, errors.Wrapf(err, "waiting for block at height %d", afterHeight+1)
	}

	rawBlocks, err := txdb.GetRawBlocks(ctx, afterHeight, 10)
	if err != nil {
		return nil, err
	}

	jsonBlocks := make([]json.HexBytes, 0, len(rawBlocks))
	for _, rb := range rawBlocks {
		jsonBlocks = append(jsonBlocks, rb)
	}
	return jsonBlocks, nil
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
