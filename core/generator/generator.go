package generator

import (
	"context"
	"time"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
)

type BlockSigner interface {
	SignBlock(context.Context, *bc.Block) (signature []byte, err error)
}

// generator produces new blocks on an interval.
type generator struct {
	// config
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
func Generate(ctx context.Context, c *protocol.Chain, s []BlockSigner, period time.Duration) {
	// This process just became leader, so it's responsible
	// for recovering after the previous leader's exit.
	recoveredBlock, recoveredSnapshot, err := c.Recover(ctx)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}

	g := &generator{
		chain:          c,
		signers:        s,
		latestBlock:    recoveredBlock,
		latestSnapshot: recoveredSnapshot,
	}

	// Check to see if we already have a pending, generated block.
	// This can happen if the leader process exits between generating
	// the block and committing the signed block to the blockchain.
	b, err := g.getPendingBlock(ctx)
	if err != nil {
		log.Fatal(ctx, err)
	}
	if b != nil && (g.latestBlock == nil || b.Height == g.latestBlock.Height+1) {
		// g.commitBlock will update g.latestBlock and g.latestSnapshot.
		_, err := g.commitBlock(ctx, b)
		if err != nil {
			log.Fatal(ctx, err)
		}
	}

	ticks := time.Tick(period)
	for {
		select {
		case <-ctx.Done():
			log.Messagef(ctx, "Deposed, Generate exiting")
			return
		case <-ticks:
			_, err := g.makeBlock(ctx)
			if err != nil {
				log.Error(ctx, err)
			}
		}
	}
}

// GetBlocks returns blocks (with heights larger than afterHeight) in
// block-height order.
func GetBlocks(ctx context.Context, c *protocol.Chain, afterHeight uint64) ([]*bc.Block, error) {
	// TODO(kr): This is not a generator function.
	// Move this to another package.
	err := c.WaitForBlock(ctx, afterHeight+1)
	if err != nil {
		return nil, errors.Wrapf(err, "waiting for block at height %d", afterHeight+1)
	}

	const q = `SELECT data FROM blocks WHERE height > $1 ORDER BY height`
	var blocks []*bc.Block
	err = pg.ForQueryRows(ctx, q, afterHeight, func(b bc.Block) {
		blocks = append(blocks, &b)
	})
	if err != nil {
		return nil, errors.Wrap(err, "querying blocks from the db")
	}
	return blocks, nil
}
