package fetch

import (
	"context"
	"time"

	"chain/errors"
	"chain/log"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/bc"
)

const getBlocksTimeout = 3 * time.Second

// Fetch runs in a loop, fetching blocks from the configured
// peer (e.g. the generator) and applying them to the local
// Chain.
//
// It returns when its context is canceled.
func Fetch(ctx context.Context, c *protocol.Chain, peerURL string) {
	// This process just became leader, so it's responsible
	// for recovering after the previous leader's exit.
	prevBlock, prevSnapshot, err := c.Recover(ctx)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Messagef(ctx, "Deposed, Fetch exiting")
			return
		default:
			var height uint64
			if prevBlock != nil {
				height = prevBlock.Height
			}

			blocks, err := getBlocks(ctx, peerURL, height)
			if err != nil {
				log.Error(ctx, err)
				continue
			}

			for _, block := range blocks {
				snapshot, err := c.ValidateBlock(ctx, prevSnapshot, prevBlock, block)
				if err != nil {
					// TODO(jackson): What do we do here? Right now, we'll busy
					// loop querying the generator over and over. Panic?
					log.Error(ctx, err)
					break
				}

				err = c.CommitBlock(ctx, block, snapshot)
				if err != nil {
					log.Error(ctx, err)
					break
				}

				prevSnapshot = snapshot
				prevBlock = block
			}
		}
	}
}

// getBlocks sends a get-blocks RPC request to another Core
// for all blocks since the highest-known one.
func getBlocks(ctx context.Context, peerURL string, height uint64) ([]*bc.Block, error) {
	ctx, cancel := context.WithTimeout(ctx, getBlocksTimeout)
	defer cancel()

	var blocks []*bc.Block
	err := rpc.Call(ctx, peerURL, "/rpc/get-blocks", height, &blocks)
	if err == context.DeadlineExceeded {
		return nil, nil
	}
	return blocks, errors.Wrap(err, "get blocks rpc")
}
