package rpcclient

import (
	"time"

	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
	"chain/errors"
	"chain/net/rpc"
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
	return rpc.Call(ctx, generatorURL, "/rpc/generator/submit", tx, nil)
}

// GetBlocks sends a get-blocks RPC request to the generator for all
// blocks since the highest-known one and adds them to the blockchain.
func GetBlocks(ctx context.Context, fc *cos.FC) error {
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

	blocks, err := getBlocks(ctx, height)
	if err == context.DeadlineExceeded {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "get blocks rpc")
	}

	for _, b := range blocks {
		err := fc.AddBlock(ctx, b)
		if err != nil {
			return errors.Wrapf(err, "applying block at height %d", b.Height)
		}
	}
	return nil
}

func getBlocks(ctx context.Context, height uint64) ([]*bc.Block, error) {
	ctx, cancel := context.WithTimeout(ctx, getBlocksTimeout)
	defer cancel()

	var blocks []*bc.Block
	err := rpc.Call(ctx, generatorURL, "/rpc/generator/get-blocks", height, &blocks)
	return blocks, errors.Wrap(err, "calling generator")
}
