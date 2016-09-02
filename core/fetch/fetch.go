package fetch

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"chain/errors"
	"chain/log"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
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

	var nfailures uint // for backoff
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
				nfailures++
				time.Sleep(backoffDur(nfailures))
				continue
			}

			prevSnapshot, prevBlock, err = applyBlocks(ctx, c, prevSnapshot, prevBlock, blocks)
			if err != nil {
				log.Error(ctx, err)
				nfailures++
				time.Sleep(backoffDur(nfailures))
				continue
			}
			nfailures = 0

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

func applyBlocks(ctx context.Context, c *protocol.Chain, snap *state.Snapshot, block *bc.Block, blocks []*bc.Block) (*state.Snapshot, *bc.Block, error) {
	for _, b := range blocks {
		ss, err := c.ValidateBlock(ctx, snap, block, b)
		if err != nil {
			// TODO(kr): this is a validation failure.
			// It's either a serious bug or an attack.
			// Do something better than just log the error
			// (in the caller above). Alert a human,
			// the security team, the legal team, the A-team,
			// somebody.
			return snap, block, err
		}

		err = c.CommitBlock(ctx, b, ss)
		if err != nil {
			return snap, block, err
		}

		snap, block = ss, b
	}
	return snap, block, nil
}

func backoffDur(n uint) time.Duration {
	if n > 33 {
		n = 33 // cap to about 10s
	}
	d := rand.Int63n(1 << n)
	return time.Duration(d)
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
