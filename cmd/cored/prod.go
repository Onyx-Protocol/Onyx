//+build prod

package main

import (
	"context"

	"chain/database/pg"
)

var prod = "yes"

func resetInDevIfRequested(db pg.DB) {}

func authLoopbackInDev(ctx context.Context) bool {
	return false
}
