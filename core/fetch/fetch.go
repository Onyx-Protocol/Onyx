package fetch

import (
	"context"

	"chain/core/rpcclient"
	"chain/log"
	"chain/protocol"
)

// Fetch runs in a loop, fetching blocks from the configured
// peer (e.g. the generator) and applying them to the local
// FC.
//
// It returns when its context is canceled.
func Fetch(ctx context.Context, fc *protocol.FC) {
	// TODO(kr): take explicit peer URL?

	// This process just became leader, so it's responsible
	// for recovering after the previous leader's exit.
	prevBlock, prevSnapshot, err := fc.Recover(ctx)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Messagef(ctx, "Deposed, Fetch exiting")
			return
		default:
			var height uint64
			if prevBlock != nil {
				height = prevBlock.Height
			}

			blocks, err := rpcclient.GetBlocks(ctx, height)
			if err != nil {
				log.Error(ctx, err)
				continue
			}

			for _, block := range blocks {
				snapshot, err := fc.ValidateBlock(ctx, prevSnapshot, prevBlock, block)
				if err != nil {
					// TODO(jackson): What do we do here? Right now, we'll busy
					// loop querying the generator over and over. Panic?
					log.Error(ctx, err)
					break
				}

				err = fc.CommitBlock(ctx, block, snapshot)
				if err != nil {
					log.Error(ctx, err)
					break
				}

				prevSnapshot = snapshot
				prevBlock = block
			}
		}
	}
}
