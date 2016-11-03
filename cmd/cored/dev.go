//+build !prod

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"chain/core"
	"chain/core/blocksigner"
	"chain/core/coreunsafe"
	"chain/core/mockhsm"
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

func hsmRegister(db pg.DB) func(*http.ServeMux, *core.API) {
	hsm := mockhsm.New(db)
	handler := &core.MockHSMHandler{MockHSM: hsm}
	return handler.Register
}

func devHSM(db pg.DB) (blocksigner.Signer, error) {
	return mockhsm.New(db), nil
}
