package fetch

import (
	"context"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"chain/core/txdb"
	"chain/errors"
	"chain/log"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
)

const getSnapshotTimeout = 60 * time.Second
const heightPollingPeriod = 3 * time.Second

var (
	generatorHeight          uint64
	generatorHeightFetchedAt time.Time
	generatorLock            sync.Mutex

	downloadingSnapshot   *Snapshot
	downloadingSnapshotMu sync.Mutex
)

func GeneratorHeight() (uint64, time.Time) {
	generatorLock.Lock()
	h := generatorHeight
	t := generatorHeightFetchedAt
	generatorLock.Unlock()
	return h, t
}

func SnapshotProgress() *Snapshot {
	downloadingSnapshotMu.Lock()
	defer downloadingSnapshotMu.Unlock()
	return downloadingSnapshot
}

// Fetch runs in a loop, fetching blocks from the configured
// peer (e.g. the generator) and applying them to the local
// Chain.
//
// It returns when its context is canceled.
// After each attempt to fetch and apply a block, it calls health
// to report either an error or nil to indicate success.
func Fetch(ctx context.Context, c *protocol.Chain, peer *rpc.Client, health func(error)) {
	// Fetch the generator height periodically.
	go pollGeneratorHeight(ctx, peer)

	if c.Height() == 0 {
		err := fetchSnapshot(ctx, peer, c.Store())
		if err != nil {
			log.Error(ctx, err)
			health(err)
		}
	}

	// This process just became leader, so it's responsible
	// for recovering after the previous leader's exit.
	prevBlock, prevSnapshot, err := c.Recover(ctx)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}

	var height uint64
	if prevBlock != nil {
		height = prevBlock.Height
	}

	blockch, errch := DownloadBlocks(ctx, peer, height+1)

	var nfailures uint
	for {
		select {
		case <-ctx.Done():
			log.Messagef(ctx, "Deposed, Fetch exiting")
			return
		case err = <-errch:
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

// DownloadBlocks starts a goroutine to download blocks from
// the given peer, starting at the given height and incrementing from there.
// It will re-attempt downloads for the next block in the network
// until it is available. It returns two channels, one for reading blocks
// and the other for reading errors. Progress will halt unless callers are
// reading from both. DownloadBlocks will continue even if it encounters errors,
// until its context is done.
func DownloadBlocks(ctx context.Context, peer *rpc.Client, height uint64) (chan *bc.Block, chan error) {
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

// Snapshot describes a snapshot being downloaded from a peer Core.
type Snapshot struct {
	Height uint64
	Size   uint64
	progressReader

	stopped   bool
	stoppedMu sync.Mutex
}

func (s *Snapshot) InProgress() bool {
	s.stoppedMu.Lock()
	defer s.stoppedMu.Unlock()
	return !s.stopped
}

func (s *Snapshot) done() {
	s.stoppedMu.Lock()
	defer s.stoppedMu.Unlock()
	s.stopped = true
}

// fetchSnapshot fetches the latest snapshot from the generator and applies it
// to the store. It should only be called on freshly configured cores--
// cores that have been operating should replay all transactions so that
// they can index them properly.
func fetchSnapshot(ctx context.Context, peer *rpc.Client, s protocol.Store) error {
	ctx, cancel := context.WithTimeout(ctx, getSnapshotTimeout)
	defer cancel()

	info := new(Snapshot)
	err := peer.Call(ctx, "/rpc/get-snapshot-info", nil, &info)
	if err != nil {
		return errors.Wrap(err, "getting snapshot info")
	}
	if info.Height == 0 {
		return nil
	}

	downloadingSnapshotMu.Lock()
	downloadingSnapshot = info
	downloadingSnapshotMu.Unlock()
	defer info.done()

	// Download the snapshot, recording our progress as we go.
	body, err := peer.CallRaw(ctx, "/rpc/get-snapshot", info.Height)
	if err != nil {
		return errors.Wrap(err, "getting snapshot")
	}
	defer body.Close()

	// Wrap the response body reader in our progress reader.
	info.progressReader.reader = body
	b, err := ioutil.ReadAll(&info.progressReader)
	if err != nil {
		return err
	}
	snapshot, err := txdb.DecodeSnapshot(b)
	if err != nil {
		return err
	}
	// Delete the snapshot issuances because we don't have any commitment
	// to them in the block. This means that Cores bootstrapping from a
	// snapshot cannot guarantee uniqueness of issuances until the max
	// issuance window has elapsed.
	snapshot.PruneIssuances(math.MaxUint64)

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
	snapshotBlock, err := getBlock(ctx, peer, info.Height, getSnapshotTimeout)
	if err != nil {
		return err
	}
	if snapshotBlock == nil {
		// Something seriously funny is still afoot.
		return errors.New("generator provided snapshot but could not provide block")
	}
	if snapshotBlock.AssetsMerkleRoot != snapshot.Tree.RootHash() {
		return errors.New("snapshot merkle root doesn't match block")
	}

	// Commit the snapshot, initial block and snapshot block.
	err = s.SaveBlock(ctx, initialBlock)
	if err != nil {
		return errors.Wrap(err, "saving the initial block")
	}
	err = s.SaveBlock(ctx, snapshotBlock)
	if err != nil {
		return errors.Wrap(err, "saving bootstrap block")
	}
	err = s.SaveSnapshot(ctx, snapshotBlock.Height, snapshot)
	return errors.Wrap(err, "saving bootstrap snaphot")
}

type progressReader struct {
	reader io.Reader
	read   uint64
}

func (r *progressReader) BytesRead() uint64 {
	return atomic.LoadUint64(&r.read)
}

func (r *progressReader) Read(b []byte) (int, error) {
	n, err := r.reader.Read(b)
	atomic.AddUint64(&r.read, uint64(n))
	return n, err
}
