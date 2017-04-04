//+build !prod

package main

import (
	"context"
	"fmt"
	"os"

	"chain/core/coreunsafe"
	"chain/database/pg"
	"chain/database/raft"
	"chain/env"
	"chain/log"
)

var (
	reset = env.String("RESET", "")
	prod  = false
)

func resetInDevIfRequested(db pg.DB, rDB *raft.Service) {
	if *reset != "" {
		os.Setenv("RESET", "")

		var err error
		ctx := context.Background()
		switch *reset {
		case "blockchain":
			err = coreunsafe.ResetBlockchain(ctx, db, rDB)
		case "everything":
			err = coreunsafe.ResetEverything(ctx, db, rDB)
		default:
			log.Fatalkv(ctx, log.KeyError, fmt.Errorf("unrecognized argument to reset: %s", *reset))
		}
		if err != nil {
			log.Fatalkv(ctx, log.KeyError, err)
		}
	}
}

func devEnableMockHSM(db pg.DB) []core.RunOption {
	return []core.RunOption{core.MockHSM(mockhsm.New(db))}
}

func devHSM(db pg.DB) (blocksigner.Signer, error) {
	return mockhsm.New(db), nil
}
