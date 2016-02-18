package generator

import (
	"chain/api/asset"
	"chain/errors"
	"chain/fedchain"
	"chain/fedchain/bc"
	"chain/log"
	"chain/net/trace/span"
	"time"

	"github.com/btcsuite/btcd/btcec"

	"golang.org/x/net/context"
)

// makeBlocks runs forever, creating a block periodically.
func makeBlocks(ctx context.Context, period time.Duration) {
	for range time.Tick(period) {
		func() {
			defer log.RecoverAndLogError(ctx)
			_, err := MakeBlock(ctx, asset.BlockKey)
			if err != nil {
				log.Error(ctx, err)
			}
		}()
	}
}

// MakeBlock creates a new bc.Block and updates the txpool/utxo state.
func MakeBlock(ctx context.Context, key *btcec.PrivateKey) (*bc.Block, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	b, err := fc.GenerateBlock(ctx, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "generate")
	}
	if len(b.Transactions) == 0 {
		return nil, nil // don't bother making an empty block
	}
	err = fedchain.SignBlock(b, key)
	if err != nil {
		return nil, errors.Wrap(err, "sign")
	}
	err = fc.AddBlock(ctx, b)
	if err != nil {
		return nil, errors.Wrap(err, "apply")
	}
	return b, nil
}
