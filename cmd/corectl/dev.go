//+build !prod

package main

import (
	"context"
	"fmt"

	"chain/core/mockhsm"
	"chain/database/sql"
)

func createBlockKeyPair(db *sql.DB, args []string) {
	if len(args) != 0 {
		fatalln("error: create-block-keypair takes no args")
	}
	ctx := context.Background()
	migrateIfMissingSchema(ctx, db)
	hsm := mockhsm.New(db)
	pub, err := hsm.Create(ctx, "block_key")
	if err != nil {
		fatalln("error:", err)
	}

	fmt.Printf("%x\n", pub.Pub)
}

func versionProdPrintln() {
	fmt.Println("production: false")
}
