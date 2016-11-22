//+build prod

package main

import (
	"net/http"

	"chain/database/pg"
)

var prod = "yes"

func resetInDevIfRequested(db pg.DB) {}

func authLoopbackInDev(req *http.Request) bool {
	return false
}
