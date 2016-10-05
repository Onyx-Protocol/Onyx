//+build prod

package main

import (
	"net/http"

	"chain/database/pg"
)

func resetInDevIfRequested(db pg.DB) {}

func initSchemaInDev(db pg.DB) {}

func authLoopbackInDev(req *http.Request) bool {
	return false
}
