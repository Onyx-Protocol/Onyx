//+build protected_db

package main

import (
	"chain/core/config"
	"chain/database/pg"
)

func init() {
	config.BuildConfig.ProtectedDB = true
}

func resetInDevIfRequested(_ pg.DB) {}
