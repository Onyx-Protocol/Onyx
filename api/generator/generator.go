package generator

import (
	"net/url"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

	"chain/api/signer"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain"
	"chain/fedchain/bc"
)

var fc *fedchain.FC

var (
	// enabled records whether the generator component has been enabled.
	enabled bool

	// remoteSigners is a slice of the addresses of the signers that
	// the generator should use.
	remoteSigners []*RemoteSigner

	localSigner *signer.Signer

	// the period at which blocks should be produced.
	blockPeriod time.Duration
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
func Init(ctx context.Context, chain *fedchain.FC, blockPubkeys []*btcec.PublicKey, nSigs int, period time.Duration, local *signer.Signer, remote []*RemoteSigner) error {
	if enabled {
		return errors.New("generator: Init called more than once")
	}
	if len(remote) == 0 && local == nil {
		return errors.New("generator: no signer configured")
	}

	fc = chain

	_, err := fc.UpsertGenesisBlock(ctx, blockPubkeys, nSigs)
	if err != nil {
		return errors.Wrap(err)
	}
	remoteSigners = remote
	localSigner = local
	blockPeriod = period
	enabled = true

	return nil
}

// Submit is an http handler for the generator submit transaction endpoint.
// Other nodes will call this endpoint to notify the generator of submitted
// transactions.
// Idempotent
func Submit(ctx context.Context, tx *bc.Tx) error {
	err := fc.AddTx(ctx, tx)
	return err
}

// GetBlocks returns blocks in block-height order.
// If afterHeight is non-nil, GetBlocks only returns
// blocks with a height larger than afterHeight.
func GetBlocks(ctx context.Context, afterHeight *uint64) ([]*bc.Block, error) {
	var startHeight uint64
	if afterHeight != nil {
		startHeight = *afterHeight + 1
	}

	q := `SELECT data FROM blocks WHERE height >= $1 ORDER BY height`
	rows, err := pg.Query(ctx, q, startHeight)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []*bc.Block

	for rows.Next() {
		var block bc.Block
		err = rows.Scan(&block)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, &block)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return blocks, nil
}
