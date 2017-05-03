//+build !no_reset

package main

import (
	"context"
	"fmt"
	"os"

	"chain/core/config"
	"chain/core/coreunsafe"
	"chain/database/pg"
	"chain/database/raft"
	"chain/env"
	"chain/log"
)

var reset = env.String("RESET", "")

func init() {
	config.BuildConfig.Reset = true
	resetIfAllowedAndRequested = func(db pg.DB, raftDB *raft.Service) {
		if *reset != "" {
			os.Setenv("RESET", "")

			var err error
			ctx := context.Background()
			switch *reset {
			case "blockchain":
				err = coreunsafe.ResetBlockchain(ctx, db, raftDB)
			case "everything":
				err = coreunsafe.ResetEverything(ctx, db, raftDB)
			default:
				log.Fatalkv(ctx, log.KeyError, fmt.Errorf("unrecognized argument to reset: %s", *reset))
			}
			if err != nil {
				log.Fatalkv(ctx, log.KeyError, err)
			}
		}
	}
}
