package generator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/log"
	"chain/metrics"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/vmutil"
)

// errTooFewSigners is returned when a block-signing attempt finds
// that not enough signers are configured for the number of
// signatures required.
var errTooFewSigners = errors.New("too few signers")

var (
	once    sync.Once
	latency *metrics.RotatingLatency
)

func recordSince(t0 time.Time) {
	// Lazily publish the expvar and initialize the rotating latency
	// histogram. We don't want to publish metrics that aren't meaningful.
	once.Do(func() {
		latency = metrics.NewRotatingLatency(5, 2*time.Second)
		metrics.PublishLatency("generator.make_block", latency)
	})
	latency.RecordSince(t0)
}

// makeBlock generates a new bc.Block, collects the required signatures
// and commits the block to the blockchain.
func (g *Generator) makeBlock(ctx context.Context) error {
	t0 := time.Now()
	defer recordSince(t0)

	g.mu.Lock()
	txs := g.pool
	g.pool = nil
	g.poolHashes = make(map[bc.Hash]bool)
	g.mu.Unlock()

	b, s, err := g.chain.GenerateBlock(ctx, g.latestBlock, g.latestSnapshot, time.Now(), txs)
	if err != nil {
		return errors.Wrap(err, "generate")
	}
	if len(b.Transactions) == 0 {
		return nil // don't bother making an empty block
	}
	err = savePendingBlock(ctx, g.db, b)
	if err != nil {
		return err
	}
	return g.commitBlock(ctx, b, s)
}

func (g *Generator) commitBlock(ctx context.Context, b *bc.Block, s *state.Snapshot) error {
	err := g.getAndAddBlockSignatures(ctx, b, g.latestBlock)
	if err != nil {
		return errors.Wrap(err, "sign")
	}

	err = g.chain.CommitAppliedBlock(ctx, b, s)
	if err != nil {
		return errors.Wrap(err, "commit")
	}

	g.latestBlock = b
	g.latestSnapshot = s
	return nil
}

func (g *Generator) getAndAddBlockSignatures(ctx context.Context, b, prevBlock *bc.Block) error {
	if prevBlock == nil && b.Height == 1 {
		return nil // no signatures needed for initial block
	}

	pubkeys, quorum, err := vmutil.ParseBlockMultiSigProgram(prevBlock.ConsensusProgram)
	if err != nil {
		return errors.Wrap(err, "parsing prevblock output script")
	}
	if len(g.signers) < quorum {
		return errTooFewSigners
	}

	hashForSig := b.Hash()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	goodSigs := make([][]byte, len(pubkeys))
	replies := make([][]byte, len(g.signers))
	done := make(chan int, len(g.signers))
	for i, signer := range g.signers {
		go getSig(ctx, signer, b, &replies[i], i, done)
	}

	nready := 0
	for i := 0; i < len(g.signers) && nready < quorum; i++ {
		sig := replies[<-done]
		if sig == nil {
			continue
		}
		k := indexKey(pubkeys, hashForSig[:], sig)
		if k >= 0 && goodSigs[k] == nil {
			goodSigs[k] = sig
			nready++
		} else if k < 0 {
			log.Printkv(ctx, "error", "invalid signature", "block", b.Hash(), "signature", sig)
		}
	}

	if nready < quorum {
		return fmt.Errorf("got %d of %d needed signatures", nready, quorum)
	}
	b.Witness = nonNilSigs(goodSigs)
	return nil
}

func indexKey(keys []ed25519.PublicKey, msg, sig []byte) int {
	for i, key := range keys {
		if ed25519.Verify(key, msg, sig) {
			return i
		}
	}
	return -1
}

func getSig(ctx context.Context, signer BlockSigner, b *bc.Block, sig *[]byte, i int, done chan int) {
	var err error
	*sig, err = signer.SignBlock(ctx, b)
	if err != nil && ctx.Err() != context.Canceled {
		log.Printkv(ctx, "error", err, "signer", signer)
	}
	done <- i
}

func nonNilSigs(a [][]byte) (b [][]byte) {
	for _, p := range a {
		if p != nil {
			b = append(b, p)
		}
	}
	return b
}

// getPendingBlock retrieves the generated, uncommitted block if it exists.
func getPendingBlock(ctx context.Context, db pg.DB) (*bc.Block, error) {
	const q = `SELECT data FROM generator_pending_block`
	var block bc.Block
	err := db.QueryRow(ctx, q).Scan(&block)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "retrieving generated pending block query")
	}
	return &block, nil
}

// savePendingBlock persists a pending, uncommitted block to the database.
// The generator should save a pending block *before* asking signers to
// sign the block.
func savePendingBlock(ctx context.Context, db pg.DB, b *bc.Block) error {
	const q = `
		INSERT INTO generator_pending_block (data) VALUES($1)
		ON CONFLICT (singleton) DO UPDATE SET data = $1;
	`
	_, err := db.Exec(ctx, q, b)
	return errors.Wrap(err, "generator_pending_block insert query")
}
