package fetch

import (
	"context"
	"io"
	"io/ioutil"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"chain/core/rpc"
	"chain/core/txdb"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
)

// SnapshotProgress describes a snapshot being downloaded from a peer Core.
type SnapshotProgress struct {
	mu               sync.Mutex
	attempt          int
	height           uint64
	size             uint64
	downloadProgress *progressReader

	stopped chan struct{}
}

// Attempt returns the how many times Core has attempted to
// download a bootstrap snapshot. If a download request times out
// or encounters any kind of validation error, it'll re-attempt the
// snapshot download a few times.
func (s *SnapshotProgress) Attempt() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attempt
}

// Height returns the blockchain height of the snapshot being
// downloaded.
func (s *SnapshotProgress) Height() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.height
}

// Progress returns the number of bytes download and the total number
// of the bytes of the snapshot.
func (s *SnapshotProgress) Progress() (downloaded, total uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.downloadProgress == nil {
		return 0, s.size
	}
	return s.downloadProgress.BytesRead(), s.size
}

// Wait blocks until the snapshot is either successfully downloaded and
// stored or the Core has given up on bootstrapping through a snapshot.
func (s *SnapshotProgress) Wait() {
	<-s.stopped
}

// BootstrapSnapshot downloads and stores the most recent snapshot from the
// provided peer. It's run when bootstrapping a new Core to an existing
// network. It should be run before invoking Chain.Recover.
func BootstrapSnapshot(ctx context.Context, c *protocol.Chain, store protocol.Store, peer *rpc.Client, health func(error)) *SnapshotProgress {
	const maxAttempts = 5

	// Return a *SnapshotProgress so that the caller can track the
	// progress of the download.
	progress := &SnapshotProgress{stopped: make(chan struct{})}
	go func() {
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			progress.mu.Lock()
			progress.attempt = attempt
			progress.downloadProgress = nil
			progress.mu.Unlock()

			err := fetchSnapshot(ctx, peer, store, progress)
			health(err)
			if err == nil {
				break
			}
			logNetworkError(ctx, err)
		}
		close(progress.stopped)
	}()
	return progress
}

// fetchSnapshot fetches the latest snapshot from the generator and applies it
// to the store. It should only be called on freshly configured cores--
// cores that have been operating should replay all transactions so that
// they can index them properly.
func fetchSnapshot(ctx context.Context, peer *rpc.Client, s protocol.Store, progress *SnapshotProgress) error {
	const getBlockTimeout = 30 * time.Second
	const readSnapshotTimeout = 30 * time.Second

	var info struct {
		Height       uint64  `json:"height"`
		Size         uint64  `json:"size"`
		BlockchainID bc.Hash `json:"blockchain_id"`
	}
	err := peer.Call(ctx, "/rpc/get-snapshot-info", nil, &info)
	if err != nil {
		return errors.Wrap(err, "getting snapshot info")
	}
	if info.Height == 0 {
		return nil
	}

	downloadCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Download the snapshot, recording our progress as we go.
	body, err := peer.CallRaw(downloadCtx, "/rpc/get-snapshot", info.Height)
	if err != nil {
		return errors.Wrap(err, "getting snapshot")
	}
	defer body.Close()

	// Wrap the response body reader in our progress reader and save
	// snapshot metadata.
	progress.mu.Lock()
	progress.size = info.Size
	progress.height = info.Height
	progress.downloadProgress = new(progressReader)
	progress.downloadProgress.reader = body
	progress.downloadProgress.setTimeout(readSnapshotTimeout, cancel)
	progress.mu.Unlock()

	b, err := ioutil.ReadAll(progress.downloadProgress)
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
	snapshot.PruneNonces(math.MaxUint64)

	// Next, get the initial block.
	initialBlock, err := getBlock(ctx, peer, 1, getBlockTimeout)
	if err != nil {
		return err
	}
	if initialBlock == nil {
		// Something seriously funny is afoot.
		return errors.New("could not get initial block from generator")
	}

	// Also get the corresponding block.
	snapshotBlock, err := getBlock(ctx, peer, info.Height, getBlockTimeout)
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

	timer           *time.Timer
	progressTimeout time.Duration
}

func (r *progressReader) setTimeout(timeout time.Duration, cancel func()) {
	r.progressTimeout = timeout
	r.timer = time.AfterFunc(timeout, cancel)
}

func (r *progressReader) BytesRead() uint64 {
	return atomic.LoadUint64(&r.read)
}

func (r *progressReader) Read(b []byte) (int, error) {
	n, err := r.reader.Read(b)

	atomic.AddUint64(&r.read, uint64(n))

	// If there's a timeout on delay between reads, then reset the timer.
	if r.timer != nil {
		r.timer.Reset(r.progressTimeout)
	}
	return n, err
}
