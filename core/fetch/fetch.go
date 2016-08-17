package fetch

import (
	"context"

	"chain/core/rpcclient"
	"chain/cos"
	"chain/log"
)

// Fetch runs in a loop, fetching blocks from the configured
// peer (e.g. the generator) and applying them to the local
// FC.
//
// It returns when its context is canceled.
func Fetch(ctx context.Context, fc *cos.FC) {
	// TODO(kr): take explicit DB and/or FC here,
	// plus maybe the peer's URL.
	for {
		select {
		case <-ctx.Done():
			log.Messagef(ctx, "Deposed, Fetch exiting")
			return
		default:
			err := rpcclient.GetBlocks(ctx, fc)
			if err != nil {
				log.Error(ctx, err)
			}
		}
	}
}
