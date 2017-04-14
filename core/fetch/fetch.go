// Package fetch implements block replication for participant
// Chain Cores.
package fetch

import (
	"context"
	"math/rand"
	"net"
	"sync"
	"time"

	"chain/core/rpc"
	"chain/errors"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc/legacy"
	"chain/protocol/state"
)

const heightPollingPeriod = 3 * time.Second

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

// Init initializes the fetch package.
func Init(ctx context.Context, peer *rpc.Client) {
	// Fetch the generator height periodically.
	go pollGeneratorHeight(ctx, peer)
}

// Fetch runs in a loop, fetching blocks from the configured
// peer (e.g. the generator) and applying them to the local
// Chain.
//
// It returns when its context is canceled.
// After each attempt to fetch and apply a block, it calls health
// to report either an error or nil to indicate success.
func Fetch(ctx context.Context, c *protocol.Chain, peer *rpc.Client, health func(error), prevBlock *legacy.Block, prevSnapshot *state.Snapshot) {
	var height uint64
	if prevBlock != nil {
		height = prevBlock.Height
	}

	blockch, errch := DownloadBlocks(ctx, peer, height+1)

	var err error
	var nfailures uint
	for {
		select {
		case <-ctx.Done():
			log.Printf(ctx, "Deposed, Fetch exiting")
			return
		case err = <-errch:
			health(err)
			logNetworkError(ctx, err)
		case b := <-blockch:
			for {
				prevSnapshot, prevBlock, err = applyBlock(ctx, c, prevSnapshot, prevBlock, b)
				if err == protocol.ErrBadBlock {
					log.Fatalkv(ctx, log.KeyError, err)
				} else if err != nil {
					// This is a serious I/O error.
					health(err)
					log.Error(ctx, err)
					nfailures++

					time.Sleep(backoffDur(nfailures))
					continue
				}
				break
			}

			height++
			health(nil)
			nfailures = 0
		}
	}
}

// DownloadBlocks starts a goroutine to download blocks from
// the given peer, starting at the given height and incrementing from there.
// It will re-attempt downloads for the next block in the network
// until it is available. It returns two channels, one for reading blocks
// and the other for reading errors. Progress will halt unless callers are
// reading from both. DownloadBlocks will continue even if it encounters errors,
// until its context is done.
func DownloadBlocks(ctx context.Context, peer *rpc.Client, height uint64) (chan *legacy.Block, chan error) {
	blockch := make(chan *legacy.Block)
	errch := make(chan error)
	go func() {
		var nfailures uint // for backoff
		var ntimeouts uint // for backoff
		for {
			select {
			case <-ctx.Done():
				close(blockch)
				close(errch)
				return
			default:
				block, err := getBlock(ctx, peer, height, timeoutBackoffDur(ntimeouts))
				if err != nil {
					errch <- err
					nfailures++
					time.Sleep(backoffDur(nfailures))
					continue
				}
				if block == nil {
					// Request time out. There might not have been any blocks published,
					// or there was a network error or it just took too long to process the
					// request.
					ntimeouts++
					continue
				}

				blockch <- block
				ntimeouts, nfailures = 0, 0
				height++
			}
		}
	}()
	return blockch, errch
}

func pollGeneratorHeight(ctx context.Context, peer *rpc.Client) {
	updateGeneratorHeight(ctx, peer)

	ticker := time.NewTicker(heightPollingPeriod)
	for {
		select {
		case <-ctx.Done():
			log.Printf(ctx, "Deposed, fetchGeneratorHeight exiting")
			ticker.Stop()
			return
		case <-ticker.C:
			updateGeneratorHeight(ctx, peer)
		}
	}
}

func updateGeneratorHeight(ctx context.Context, peer *rpc.Client) {
	gh, err := getHeight(ctx, peer)
	if err != nil {
		logNetworkError(ctx, err)
		return
	}

	generatorLock.Lock()
	defer generatorLock.Unlock()
	generatorHeight = gh
	generatorHeightFetchedAt = time.Now()
}

func applyBlock(ctx context.Context, c *protocol.Chain, prevSnap *state.Snapshot, prev *legacy.Block, block *legacy.Block) (*state.Snapshot, *legacy.Block, error) {
	err := c.ValidateBlock(block, prev)
	if err != nil {
		return prevSnap, prev, err
	}
	snap, err := c.ApplyValidBlock(block)
	if err != nil {
		return prevSnap, prev, err
	}
	err = c.CommitAppliedBlock(ctx, block, snap)
	if err != nil {
		return prevSnap, prev, err
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

func timeoutBackoffDur(n uint) time.Duration {
	const baseTimeout = 3 * time.Second
	if n > 4 {
		n = 4 // cap to extra 16s
	}
	d := rand.Int63n(int64(time.Second) * (1 << n))
	return baseTimeout + time.Duration(d)
}

// getBlock sends a get-block RPC request to another Core
// for the next block.
func getBlock(ctx context.Context, peer *rpc.Client, height uint64, timeout time.Duration) (*legacy.Block, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var block *legacy.Block
	err := peer.Call(ctx, "/rpc/get-block", height, &block)
	if ctx.Err() == context.DeadlineExceeded {
		return nil, nil
	}
	return block, errors.Wrap(err, "get blocks rpc")
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

func logNetworkError(ctx context.Context, err error) {
	if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
		log.Printf(ctx, "%s", err.Error())
	} else {
		log.Error(ctx, err)
	}
}
