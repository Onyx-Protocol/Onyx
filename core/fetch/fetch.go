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

	var height uint64
	if prevBlock != nil {
		height = prevBlock.Height
	}

	dctx, dcancel := context.WithCancel(ctx)
	blockch, errch := downloadBlocks(dctx, peer, height+1)

	var nfailures uint
	for {
		select {
		case <-ctx.Done():
			log.Messagef(ctx, "Deposed, Fetch exiting")
			dcancel()
			return
		case err := <-errch:
			health(err)
			log.Error(ctx, err)
		case b := <-blockch:
			for {
				prevSnapshot, prevBlock, err = applyBlock(ctx, c, prevSnapshot, prevBlock, b)
				if err == protocol.ErrBadBlock {
					log.Fatal(ctx, log.KeyError, err)
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

func downloadBlocks(ctx context.Context, peer *rpc.Client, height uint64) (chan *bc.Block, chan error) {
	blockch := make(chan *bc.Block)
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

func applyBlock(ctx context.Context, c *protocol.Chain, prevSnap *state.Snapshot, prev *bc.Block, block *bc.Block) (*state.Snapshot, *bc.Block, error) {
	snap, err := c.ValidateBlock(ctx, prevSnap, prev, block)
	if err != nil {
		return prevSnap, prev, err
	}

	err = c.CommitBlock(ctx, block, snap)
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
func getBlock(ctx context.Context, peer *rpc.Client, height uint64, timeout time.Duration) (*bc.Block, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var block *bc.Block
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

	// Next, get the initial block.
	initialBlock, err := getBlock(ctx, peer, 1, getSnapshotTimeout)
	if err != nil {
		return err
	}
	if initialBlock == nil {
		// Something seriously funny is afoot.
		return errors.New("could not get initial block from generator")
	}

	// Also get the corresponding block.
	snapshotBlock, err := getBlock(ctx, peer, snapResp.Height, getSnapshotTimeout)
	if err != nil {
		return err
	}
	if snapshotBlock == nil {
		// Something seriously funny is still afoot.
		return errors.New("generator provided snapshot but could not provide block")
	}

	// Commit everything to the database. The order here is important. The
	// snapshot needs to be last. If there's a failure at any point, the
	// Core will end up recovering back to the empty blockchain state.
	err = s.SaveBlock(ctx, initialBlock)
	if err != nil {
		return errors.Wrap(err, "saving the initial block")
	}
	err = s.SaveBlock(ctx, snapshotBlock)
	if err != nil {
		return errors.Wrap(err, "saving bootstrap block")
	}
	const snapQ = `
		INSERT INTO snapshots (height, data) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	_, err = db.Exec(ctx, snapQ, snapResp.Height, snapResp.Data)
	return errors.Wrap(err, "saving bootstrap snaphot")
}
