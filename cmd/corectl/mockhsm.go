//+build !no_mockhsm

package main

import (
	"context"
	"fmt"

	"chain/core/mockhsm"
	"chain/core/rpc"
)

func createBlockKeyPair(client *rpc.Client, args []string) {
	if len(args) != 0 {
		fatalln("error: create-block-keypair takes no args")
	}
	var pub mockhsm.Pub
	err := client.Call(context.Background(), "/mockhsm/create-block-key", nil, &pub)
	if err != nil {
		fatalln("error:", err)
	}
	fmt.Printf("%x\n", pub.Pub)
}
