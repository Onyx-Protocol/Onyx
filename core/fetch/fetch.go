package fetch

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"chain/database/sql"
	"chain/errors"
	"chain/log"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
)

const getSnapshotTimeout = 10 * time.Second
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

// Fetch runs in a loop, fetching blocks from the configured
// peer (e.g. the generator) and applying them to the local
// Chain.
//
// It returns when its context is canceled.
// After each attempt to fetch and apply a block, it calls health
// to report either an error or nil to indicate success.
func Fetch(ctx context.Context, c *protocol.Chain, peer *rpc.Client, health func(error)) {
	// This process just became leader, so it's responsible
	// for recovering after the previous leader's exit.
	prevBlock, prevSnapshot, err := c.Recover(ctx)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}

	// Fetch the generator height periodically.
	go pollGeneratorHeight(ctx, peer)

	var ntimeouts uint // for backoff
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

			blocks, err := getBlocks(ctx, peer, height, timeoutBackoffDur(ntimeouts))
			if err != nil {
				health(err)
				log.Error(ctx, err)
				nfailures++
				time.Sleep(backoffDur(nfailures))
				continue
			}
			if len(blocks) == 0 {
				// Request time out. There might not have been any blocks published,
				// or there was a network error or it just took too long to process the
				// request.
				ntimeouts++
				continue
			}

			prevSnapshot, prevBlock, err = applyBlocks(ctx, c, prevSnapshot, prevBlock, blocks)
			if err != nil {
				health(err)
				log.Error(ctx, err)
				nfailures++
				time.Sleep(backoffDur(nfailures))
				continue
			}
			health(nil)
			nfailures, ntimeouts = 0, 0
		}
	}
}

func pollGeneratorHeight(ctx context.Context, peer *rpc.Client) {
	updateGeneratorHeight(ctx, peer)

	ticker := time.NewTicker(heightPollingPeriod)
	for {
		select {
		case <-ctx.Done():
			log.Messagef(ctx, "Deposed, fetchGeneratorHeight exiting")
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
		log.Error(ctx, err)
		return
	}

	generatorLock.Lock()
	defer generatorLock.Unlock()
	generatorHeight = gh
	generatorHeightFetchedAt = time.Now()
}

func applyBlocks(ctx context.Context, c *protocol.Chain, snap *state.Snapshot, block *bc.Block, blocks []*bc.Block) (*state.Snapshot, *bc.Block, error) {
	for _, b := range blocks {
		ss, err := c.ValidateBlock(ctx, snap, block, b)
		if err != nil {
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

func timeoutBackoffDur(n uint) time.Duration {
	const baseTimeout = 3 * time.Second
	if n > 4 {
		n = 4 // cap to extra 16s
	}
	d := rand.Int63n(int64(time.Second) * (1 << n))
	return baseTimeout + time.Duration(d)
}

// getBlocks sends a get-blocks RPC request to another Core
// for all blocks since the highest-known one.
func getBlocks(ctx context.Context, peer *rpc.Client, height uint64, timeout time.Duration) ([]*bc.Block, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var blocks []*bc.Block
	err := peer.Call(ctx, "/rpc/get-blocks", height, &blocks)
	if ctx.Err() == context.DeadlineExceeded {
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

// Snapshot fetches the latest snapshot from the generator and applies it to this
// core's snapshot set. It should only be called on freshly configured cores--
// cores that have been operating should replay all transactions so that they can
// index them properly.
func Snapshot(ctx context.Context, peer *rpc.Client, s protocol.Store, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, getSnapshotTimeout)
	defer cancel()

	var snapResp struct {
		Data   []byte
		Height uint64
	}
	err := peer.Call(ctx, "/rpc/get-snapshot", nil, &snapResp)
	if err != nil {
		return err
	}

	const snapQ = `
		INSERT INTO snapshots (height, data) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	_, err = db.Exec(ctx, snapQ, snapResp.Height, snapResp.Data)
	if err != nil {
		return err
	}

	// Next, get the genesis block.
	blocks, err := getBlocks(ctx, peer, 0, getSnapshotTimeout)
	if err != nil {
		return err
	}

	if len(blocks) < 1 {
		// Something seriously funny is afoot.
		return errors.New("could not get initial block from generator")
	}

	err = s.SaveBlock(ctx, blocks[0])

	// Also get the corresponding block.
	blocks, err = getBlocks(ctx, peer, snapResp.Height-1, getSnapshotTimeout) // because we get the NEXT block
	if err != nil {
		return err
	}

	if len(blocks) < 1 {
		// Something seriously funny is still afoot.
		return errors.New("generator provided snapshot but could not provide block")
	}

	return s.SaveBlock(ctx, blocks[0])
}
