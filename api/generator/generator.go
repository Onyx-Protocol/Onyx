package generator

import (
	"net/url"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/signer"
	"chain/errors"
)

var (
	// enabled records whether the generator component has been enabled.
	enabled bool

	// remoteSigners is a slice of the addresses of the signers that
	// the generator should use.
	remoteSigners []RemoteSigner

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
// It will attempt to create one block per period.
// The signers in remote will be contacted to sign each block.
// The local signer, if non-nil, will also sign each block.
func Init(ctx context.Context, period time.Duration, local *signer.Signer, remote []RemoteSigner) error {
	if enabled {
		return errors.New("generator: Init called more than once")
	}
	if len(remote) == 0 && local == nil {
		return errors.New("generator: no signer configured")
	}

	_, err := asset.UpsertGenesisBlock(ctx)
	if err != nil {
		return errors.Wrap(err)
	}
	remoteSigners = remote
	localSigner = local
	blockPeriod = period
	enabled = true
	go asset.MakeBlocks(ctx, blockPeriod)
	return nil
}
