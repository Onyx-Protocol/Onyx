//+build !disable_reset

package main

import (
	"chain/core/coreunsafe"
	"chain/database/pg"
	"chain/env"
	"chain/log"
	"context"
	"fmt"
	"os"
)

var (
	reset = env.String("RESET", "")
)

func resetInDevIfRequested(db pg.DB) {
	if *reset != "" {
		os.Setenv("RESET", "")

		var err error
		ctx := context.Background()
		switch *reset {
		case "blockchain":
			err = coreunsafe.ResetBlockchain(ctx, db)
		case "everything":
			err = coreunsafe.ResetEverything(ctx, db)
		default:
			log.Fatalkv(ctx, log.KeyError, fmt.Errorf("unrecognized argument to reset: %s", *reset))
		}
		if err != nil {
			log.Fatalkv(ctx, log.KeyError, err)
		}
	}
}
