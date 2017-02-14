//+build prod

package main

import (
	"errors"
	"net/http"

	"chain/core"
	"chain/core/blocksigner"
	"chain/database/pg"
)

var prod = true

func resetInDevIfRequested(db pg.DB) {}

func authLoopbackInDev(req *http.Request) bool {
	return false
}

func hsmRegister(_ pg.DB) func(*http.ServeMux, *core.API) {
	return nil
}

func devHSM(_ pg.DB) (blocksigner.Signer, error) {
	return nil, errors.New("cannot use mockhsm in production, must configure block hsm url")
}
