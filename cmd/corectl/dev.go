//+build !prod

package main

import (
	"context"
	"fmt"

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

func versionProdPrintln() {
	fmt.Println("production: false")
}
