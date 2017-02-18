//+build !prod

package main

import (
	"context"
	"fmt"

	"chain/core/coreunsafe"
	"chain/core/mockhsm"
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

func createBlockKeyPair(db *sql.DB, _ *raft.Service, args []string) {
	if len(args) != 0 {
		fatalln("error: create-block-keypair takes no args")
	}

	hsm := mockhsm.New(db)
	ctx := context.Background()
	pub, err := hsm.Create(ctx, "block_key")
	if err != nil {
		fatalln("error:", err)
	}

	fmt.Printf("%x\n", pub.Pub)
}
