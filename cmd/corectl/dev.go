//+build !prod

package main

import (
	"context"
	"fmt"

	"chain/core/coreunsafe"
	"chain/core/mockhsm"
	"chain/core/rpc"
	"chain/database/sql"
)

func reset(db *sql.DB, args []string) {
	if len(args) != 0 {
		fatalln("error: reset takes no args")
	}

	// TODO(tessr): TLS everywhere?
	client := &rpc.Client{
		BaseURL: *coreURL,
	}

	req := map[string]bool{
		"Everything": true,
	}

	ctx := context.Background()
	err := client.Call(ctx, "/reset", req, nil)
	if err != nil {
		fatalln("rpc error:", err)
	}
}

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
