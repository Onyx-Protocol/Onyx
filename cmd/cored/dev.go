//+build !prod

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"chain/core/coreunsafe"
	"chain/database/pg"
	"chain/env"
	"chain/log"
)

var (
	reset = env.String("RESET", "")
	prod  = "no"
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
			log.Fatal(ctx, log.KeyError, fmt.Errorf("unrecognized argument to reset: %s", *reset))
		}
		if err != nil {
			log.Fatal(ctx, log.KeyError, err)
		}
	}
}

func authLoopbackInDev(req *http.Request) bool {
	// Allow connections from the local host.
	a, err := net.ResolveTCPAddr("tcp", req.RemoteAddr)
	return err == nil && a.IP.IsLoopback()
}
