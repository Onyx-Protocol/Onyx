//+build disable_reset

package main

import (
	"chain/core/config"
	"chain/database/pg"
)

func init() {
	config.Reset = false
}

func resetInDevIfRequested(db pg.DB) {}
