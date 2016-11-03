//+build !prod

package main

import (
	"context"

	"chain/core/coreunsafe"
	"chain/database/raft"
	"chain/database/sql"
)

func reset(db *sql.DB, rDB *raft.Service, args []string) {
	if len(args) != 0 {
		fatalln("error: reset takes no args")
	}

	ctx := context.Background()
	err := coreunsafe.ResetEverything(ctx, db, rDB)
	if err != nil {
		fatalln("error:", err)
	}
}
