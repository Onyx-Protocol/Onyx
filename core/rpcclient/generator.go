package rpcclient

import (
	"context"
	"time"

	"chain/errors"
	"chain/net/rpc"
	"chain/protocol/bc"
)

const (
	getBlocksTimeout = 3 * time.Second
)

// Submit sends a submit RPC request to the generator for inclusion of
// a new transaction in the next block.
func Submit(ctx context.Context, tx *bc.Tx) error {
	if generatorURL == "" {
		return ErrNoGenerator
	}
	return rpc.Call(ctx, generatorURL, "/rpc/submit", tx, nil)
}

// GetBlocks sends a get-blocks RPC request to the generator
// for all blocks since the highest-known one.
func GetBlocks(ctx context.Context, height uint64) ([]*bc.Block, error) {
	if generatorURL == "" {
		return nil, ErrNoGenerator
	}

	ctx, cancel := context.WithTimeout(ctx, getBlocksTimeout)
	defer cancel()

	var blocks []*bc.Block
	err := rpc.Call(ctx, generatorURL, "/rpc/get-blocks", height, &blocks)
	if err == context.DeadlineExceeded {
		return nil, nil
	}
	return blocks, errors.Wrap(err, "get blocks rpc")
}
