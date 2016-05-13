package rpcclient

import (
	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
	"chain/errors"
	"chain/net/rpc"
)

// Submit sends a submit RPC request to the generator for inclusion of
// a new transaction in the next block.
func Submit(ctx context.Context, tx *bc.Tx) error {
	if generatorURL == "" {
		return ErrNoGenerator
	}
	return rpc.Call(ctx, generatorURL, "/rpc/generator/submit", tx, nil)
}

// GetBlocks sends a get-blocks RPC request to the generator for all
// blocks since the highest-known one and adds them to the blockchain.
func GetBlocks(ctx context.Context) error {
	if generatorURL == "" {
		return ErrNoGenerator
	}

	latestBlock, err := fc.LatestBlock(ctx)
	if err != nil && errors.Root(err) != cos.ErrNoBlocks {
		return errors.Wrap(err, "looking up last-known block")
	}

	var height uint64
	if latestBlock != nil {
		height = latestBlock.Height
	}

	var blocks []*bc.Block
	err = rpc.Call(ctx, generatorURL, "/rpc/generator/get-blocks", height, &blocks)
	if err != nil {
		return errors.Wrap(err, "calling generator")
	}

	for _, b := range blocks {
		err := fc.AddBlock(ctx, b)
		if err != nil {
			return errors.Wrapf(err, "applying block at height %d", b.Height)
		}
	}

	return nil
}
