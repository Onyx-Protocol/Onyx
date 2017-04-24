// Package generator implements the Chain Core generator.
//
// A Chain Core configured as a generator produces new blocks
// on an interval.
package generator

import (
	"context"
	"sync"
	"time"

	"chain/database/pg"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
)

// A BlockSigner signs blocks.
type BlockSigner interface {
	// SignBlock returns an ed25519 signature over the block's sighash.
	// See also the Chain Protocol spec for the complete required behavior
	// of a block signer.
	SignBlock(ctx context.Context, marshalledBlock []byte) (signature []byte, err error)
}

// Generator collects pending transactions and produces new blocks on
// an interval.
type Generator struct {
	// config
	db      pg.DB
	chain   *protocol.Chain
	signers []BlockSigner

	mu         sync.Mutex
	pool       []*legacy.Tx // in topological order
	poolHashes map[bc.Hash]bool
}

// New creates and initializes a new Generator.
func New(
	c *protocol.Chain,
	s []BlockSigner,
	db pg.DB,
) *Generator {
	return &Generator{
		db:         db,
		chain:      c,
		signers:    s,
		poolHashes: make(map[bc.Hash]bool),
	}
}

// PendingTxs returns all of the pendings txs that will be
// included in the generator's next block.
func (g *Generator) PendingTxs() []*legacy.Tx {
	g.mu.Lock()
	defer g.mu.Unlock()

	txs := make([]*legacy.Tx, len(g.pool))
	copy(txs, g.pool)
	return txs
}

// Submit adds a new pending tx to the pending tx pool.
func (g *Generator) Submit(ctx context.Context, tx *legacy.Tx) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.poolHashes[tx.ID] {
		return nil
	}

	g.poolHashes[tx.ID] = true
	g.pool = append(g.pool, tx)
	return nil
}

// Generate runs in a loop, making one new block
// every block period. It returns when its context
// is canceled.
// After each attempt to make a block, it calls health
// to report either an error or nil to indicate success.
func (g *Generator) Generate(
	ctx context.Context,
	period time.Duration,
	health func(error),
) {
	ticks := time.Tick(period)
	for {
		select {
		case <-ctx.Done():
			log.Printf(ctx, "Deposed, Generate exiting")
			return
		case <-ticks:
			err := g.makeBlock(ctx)
			health(err)
			if err != nil {
				log.Error(ctx, err)
			}
		}
	}
}
