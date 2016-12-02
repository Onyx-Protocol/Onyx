package txbuilder

import (
	"context"

	"chain/core/rpc"
	"chain/errors"
	"chain/protocol/bc"
)

// Submitter submits a transaction to the generator so that it may
// be confirmed in a block.
type Submitter interface {
	Submit(ctx context.Context, tx *bc.Tx) error
}

// RemoteGenerator implements the Submitter interface and submits the
// transaction to a remote generator.
// TODO(jackson): This implementation maybe belongs elsewhere.
type RemoteGenerator struct {
	Peer *rpc.Client
}

func (rg *RemoteGenerator) Submit(ctx context.Context, tx *bc.Tx) error {
	err := rg.Peer.Call(ctx, "/rpc/submit", tx, nil)
	err = errors.Wrap(err, "generator transaction notice")
	return err
}
