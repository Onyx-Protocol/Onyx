//+build !no_mockhsm

package main

import (
	"context"
	"fmt"

	"chain/core/mockhsm"
	"chain/database/sql"
)

func createBlockKeyPair(_ *sql.DB, args []string) {
	if len(args) != 0 {
		fatalln("error: create-block-keypair takes no args")
	}
	var pub mockhsm.Pub
	client := mustRPCClient()
	err := client.Call(context.Background(), "/mockhsm/create-block-key", nil, &pub)
	if err != nil {
		fatalln("rpc error:", err)
	}
	fmt.Printf("%x\n", pub.Pub)
}
