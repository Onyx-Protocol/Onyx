//+build !protected_db

package main

import (
	"chain/core/coreunsafe"
	"chain/database/sql"
	"context"
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
