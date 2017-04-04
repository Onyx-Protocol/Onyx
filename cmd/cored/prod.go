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
