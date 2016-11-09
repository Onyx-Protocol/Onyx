package core

import (
	"context"
	"encoding/json"
	"net/http"

	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/net/http/httpjson"
	"chain/protocol/bc"
)

// getBlockRPC returns the block at the requested height.
// If successful, it always returns at least one block,
// waiting if necessary until one is created.
// It is an error to request blocks very far in the future.
func (h *Handler) getBlockRPC(ctx context.Context, height uint64) (chainjson.HexBytes, error) {
	err := <-h.Chain.BlockSoonWaiter(ctx, height)
	if err != nil {
		return nil, errors.Wrapf(err, "waiting for block at height %d", height)
	}

	rawBlock, err := h.Store.GetRawBlock(ctx, height)
	if err != nil {
		return nil, err
	}

	return rawBlock, nil
}

// getBlocksRPC -- DEPRECATED: use getBlock instead
func (h *Handler) getBlocksRPC(ctx context.Context, afterHeight uint64) ([]chainjson.HexBytes, error) {
	block, err := h.getBlockRPC(ctx, afterHeight+1)
	if err != nil {
		return nil, err
	}

	return []chainjson.HexBytes{block}, nil
}

type snapshotInfoResp struct {
	Height       uint64  `json:"height"`
	Size         uint64  `json:"size"`
	BlockchainID bc.Hash `json:"blockchain_id"`
}

func (h *Handler) getSnapshotInfoRPC(ctx context.Context) (resp snapshotInfoResp, err error) {
	// TODO(jackson): cache latest snapshot and its height & size in-memory.
	resp.Height, resp.Size, err = h.Store.LatestSnapshotInfo(ctx)
	resp.BlockchainID = h.Config.BlockchainID
	return resp, err
}

// getSnapshotRPC returns the raw protobuf snapshot at the provided height.
// Non-generators can call this endpoint to get raw data
// that they can use to populate their own snapshot table.
//
// This handler doesn't use the httpjson.Handler format so that it can return
// raw protobuf bytes on the wire.
func (h *Handler) getSnapshotRPC(rw http.ResponseWriter, req *http.Request) {
	if h.Config == nil {
		alwaysError(errUnconfigured).ServeHTTP(rw, req)
		return
	}

	var height uint64
	err := json.NewDecoder(req.Body).Decode(&height)
	if err != nil {
		WriteHTTPError(req.Context(), rw, httpjson.ErrBadRequest)
		return
	}

	data, err := h.Store.GetSnapshot(req.Context(), height)
	if err != nil {
		WriteHTTPError(req.Context(), rw, err)
		return
	}
	rw.Header().Set("Content-Type", "application/x-protobuf")
	rw.Write(data)
}
