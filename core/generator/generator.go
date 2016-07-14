package generator

import (
	"net/url"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

	"chain/core/signer"
	"chain/cos"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/errors"
	"chain/log"
)

var fc *cos.FC

var (
	// enabled records whether the generator component has been enabled.
	enabled bool

	// remoteSigners is a slice of the addresses of the signers that
	// the generator should use.
	remoteSigners []*RemoteSigner

	localSigner *signer.Signer

	// the period at which blocks should be produced.
	blockPeriod time.Duration

	// the keys used for block scripts
	blockKeys []*btcec.PublicKey

	// the number of signatures required for block scripts
	sigsRequired int
)

type RemoteSigner struct {
	URL *url.URL
	Key *btcec.PublicKey
}

// Enabled returns whether the generator is enabled on the node.
func Enabled() bool {
	return enabled
}

// Init initializes and enables the generator component of the node.
// It must be called before any other functions in this package.
// The signers in remote will be contacted to sign each block.
// The local signer, if non-nil, will also sign each block.
//
// TODO(bobg): Remove the period parameter, since this function no
// longer launches a make-blocks goroutine.  (But for now it's used to
// initialize a package-private copy of that value for use in
// GetSummary.)
func Init(ctx context.Context, chain *cos.FC, blockPubkeys []*btcec.PublicKey, nSigs int, period time.Duration, local *signer.Signer, remote []*RemoteSigner) error {
	if len(remote) == 0 && local == nil {
		return errors.New("generator: no signer configured")
	}

	fc = chain
	blockKeys = blockPubkeys
	sigsRequired = nSigs

	remoteSigners = remote
	localSigner = local
	blockPeriod = period
	enabled = true

	return nil
}

// Generate runs in a loop, making one new block
// every block period. It returns when its context
// is canceled.
func Generate(ctx context.Context) {
	err := UpsertGenesisBlock(ctx)
	if err != nil {
		panic(err)
	}

	ticks := time.Tick(blockPeriod)
	for {
		select {
		case <-ctx.Done():
			log.Messagef(ctx, "Deposed, Generate exiting")
			return
		case <-ticks:
			_, err := MakeBlock(ctx)
			if err != nil {
				log.Error(ctx, err)
			}
		}
	}
}

// UpsertGenesisBlock upserts a genesis block using
// the keys and signatures required provided to Init.
func UpsertGenesisBlock(ctx context.Context) error {
	_, err := fc.UpsertGenesisBlock(ctx, blockKeys, sigsRequired)
	return errors.Wrap(err)
}

// Submit is an http handler for the generator submit transaction endpoint.
// Other nodes will call this endpoint to notify the generator of submitted
// transactions.
// Idempotent
func Submit(ctx context.Context, tx *bc.Tx) error {
	err := fc.AddTx(ctx, tx)
	return err
}

// GetBlocks returns blocks (with heights larger than afterHeight) in
// block-height order.
func GetBlocks(ctx context.Context, afterHeight uint64) ([]*bc.Block, error) {
	err := fc.WaitForBlock(ctx, afterHeight+1)
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
