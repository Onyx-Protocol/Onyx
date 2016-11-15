//+build !prod

package main

import (
	"context"

	"chain/core/coreunsafe"
	"chain/database/sql"
)

func reset(db *sql.DB, args []string) {
	if len(args) != 0 {
		fatalln("error: reset takes no args")
	}

	ctx := context.Background()
	err := coreunsafe.ResetEverything(ctx, db) // #nosec
	if err != nil {
		fatalln("error:", err)
	}
}
