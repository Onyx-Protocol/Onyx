//+build !prod

package main

import (
	"context"
	"os"

	"chain/core"
	"chain/database/pg"
	"chain/env"
)

var reset = env.Bool("RESET", false)

func requireSecretInProd(secret string) {}

func resetInDevIfRequested(db pg.DB) {
	if *reset {
		os.Setenv("RESET", "false")
		ctx := context.Background()
		core.Reset(ctx, db)
	}
}
