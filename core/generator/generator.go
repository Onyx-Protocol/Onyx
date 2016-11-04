// Package generator implements the Chain Core generator.
//
// A Chain Core configured as a generator produces new blocks
// on an interval.
package generator

import (
	"context"
	"time"

	"chain/database/pg"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/validation"
)

// A BlockSigner signs blocks.
type BlockSigner interface {
	// SignBlock returns an ed25519 signature over the block's sighash.
	// See also the Chain Protocol spec for the complete required behavior
	// of a block signer.
	SignBlock(context.Context, *bc.Block) (signature []byte, err error)
}

// generator produces new blocks on an interval.
type generator struct {
	// config
	db      pg.DB
	chain   *protocol.Chain
	signers []BlockSigner

	// latestBlock and latestSnapshot are current as long as this
	// process remains the leader process. If the process is demoted,
	// generator.Generate() should return and this struct should be
	// garbage collected.
	latestBlock    *bc.Block
	latestSnapshot *state.Snapshot
}

// Generate runs in a loop, making one new block
// every block period. It returns when its context
// is canceled.
// After each attempt to make a block, it calls health
// to report either an error or nil to indicate success.
func Generate(
	ctx context.Context,
	c *protocol.Chain,
	s []BlockSigner,
	db pg.DB,
	period time.Duration,
	health func(error),
) {
	// This process just became leader, so it's responsible
	// for recovering after the previous leader's exit.
	recoveredBlock, recoveredSnapshot, err := c.Recover(ctx)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}

	g := &generator{
		db:             db,
		chain:          c,
		signers:        s,
		latestBlock:    recoveredBlock,
		latestSnapshot: recoveredSnapshot,
	}

	// Check to see if we already have a pending, generated block.
	// This can happen if the leader process exits between generating
	// the block and committing the signed block to the blockchain.
	b, err := getPendingBlock(ctx, g.db)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}
	if b != nil && (g.latestBlock == nil || b.Height == g.latestBlock.Height+1) {
		s := state.Copy(g.latestSnapshot)
		err := validation.ApplyBlock(s, b)
		if err != nil {
			log.Fatal(ctx, log.KeyError, err)
		}

		// g.commitBlock will update g.latestBlock and g.latestSnapshot.
		err = g.commitBlock(ctx, b, s)
		if err != nil {
			log.Fatal(ctx, log.KeyError, err)
		}
	}

	ticks := time.Tick(period)
	for {
		select {
		case <-ctx.Done():
			log.Messagef(ctx, "Deposed, Generate exiting")
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
