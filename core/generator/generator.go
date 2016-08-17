package generator

import (
	"context"
	"net/url"
	"time"

	"chain/core/blocksigner"
	"chain/cos"
	"chain/cos/bc"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/errors"
	"chain/log"
)

// Config encapsulates generator configuration options.
type Config struct {
	RemoteSigners []*RemoteSigner
	LocalSigner   *blocksigner.Signer
	BlockPeriod   time.Duration
	BlockKeys     []ed25519.PublicKey // keys for block scripts
	SigsRequired  int                 // sigs required for block scripts
	FC            *cos.FC
}

// Generator produces new blocks on an interval.
type Generator struct {
	Config
	// TODO(jackson): Add current state tree.
}

// RemoteSigner defines the address and public key of another Core
// that may sign blocks produced by this generator.
type RemoteSigner struct {
	URL *url.URL
	Key ed25519.PublicKey
}

// Generate runs in a loop, making one new block
// every block period. It returns when its context
// is canceled.
func Generate(ctx context.Context, config Config) {
	g := &Generator{
		Config: config,
	}

	err := g.UpsertGenesisBlock(ctx)
	if err != nil {
		panic(err)
	}

	ticks := time.Tick(g.BlockPeriod)
	for {
		select {
		case <-ctx.Done():
			log.Messagef(ctx, "Deposed, Generate exiting")
			return
		case <-ticks:
			_, err := g.MakeBlock(ctx)
			if err != nil {
				log.Error(ctx, err)
			}
		}
	}
}

// UpsertGenesisBlock upserts a genesis block using
// the keys and signatures required.
func (g *Config) UpsertGenesisBlock(ctx context.Context) error {
	_, err := g.FC.UpsertGenesisBlock(ctx, g.BlockKeys, g.SigsRequired, time.Now())
	return errors.Wrap(err)
}

// Submit is an http handler for the generator submit transaction endpoint.
// Other nodes will call this endpoint to notify the generator of submitted
// transactions.
func (g *Config) Submit(ctx context.Context, tx *bc.Tx) error {
	err := g.FC.AddTx(ctx, tx)
	return err
}

// GetBlocks returns blocks (with heights larger than afterHeight) in
// block-height order.
func (g *Config) GetBlocks(ctx context.Context, afterHeight uint64) ([]*bc.Block, error) {
	err := g.FC.WaitForBlock(ctx, afterHeight+1)
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
