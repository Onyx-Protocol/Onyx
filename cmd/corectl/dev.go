//+build !prod

package main

import (
	"context"

	"chain-stealth/core/coreunsafe"
	"chain-stealth/database/sql"
)

func reset(db *sql.DB, args []string) {
	if len(args) != 0 {
		fatalln("error: reset takes no args")
	}

	ctx := context.Background()
	err := coreunsafe.ResetEverything(ctx, db)
	if err != nil {
		fatalln("error:", err)
	}
}
