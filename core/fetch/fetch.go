package fetch

import (
	"context"
	"sync"
	"time"

	"chain/errors"
	"chain/log"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/bc"
)

const getBlocksTimeout = 3 * time.Second

var (
	generatorHeight          uint64
	generatorHeightFetchedAt time.Time
	generatorLock            sync.Mutex
)

func GeneratorHeight() (uint64, time.Time) {
	generatorLock.Lock()
	h := generatorHeight
	t := generatorHeightFetchedAt
	generatorLock.Unlock()
	return h, t
}

// Fetch runs in a loop, fetching blocks from the configured
// peer (e.g. the generator) and applying them to the local
// Chain.
//
// It returns when its context is canceled.
func Fetch(ctx context.Context, c *protocol.Chain, peer *rpc.Client) {
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

			blocks, err := getBlocks(ctx, peer, height)
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

			gh, err := getHeight(ctx, peer)
			if err != nil {
				log.Error(ctx, err)
			} else {
				generatorLock.Lock()
				generatorHeight = gh
				generatorHeightFetchedAt = time.Now()
				generatorLock.Unlock()
			}
		}
	}
}

// getBlocks sends a get-blocks RPC request to another Core
// for all blocks since the highest-known one.
func getBlocks(ctx context.Context, peer *rpc.Client, height uint64) ([]*bc.Block, error) {
	ctx, cancel := context.WithTimeout(ctx, getBlocksTimeout)
	defer cancel()

	var blocks []*bc.Block
	err := peer.Call(ctx, "/rpc/get-blocks", height, &blocks)
	if err == context.DeadlineExceeded {
		return nil, nil
	}
	return blocks, errors.Wrap(err, "get blocks rpc")
}

// getHeight sends a get-height RPC request to another Core for
// the latest height that that peer knows about.
func getHeight(ctx context.Context, peer *rpc.Client) (uint64, error) {
	var resp map[string]uint64
	err := peer.Call(ctx, "/rpc/block-height", nil, &resp)
	if err != nil {
		return 0, errors.Wrap(err, "could not get remote block height")
	}
	h, ok := resp["block_height"]
	if !ok {
		return 0, errors.New("unexpected response from generator")
	}

	return h, nil
}
