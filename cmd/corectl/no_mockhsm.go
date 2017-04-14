//+build no_mockhsm

package main

import "chain/core/rpc"

func createBlockKeyPair(*rpc.Client, []string) {
	fatalln("error: create-block-keypair disabled in no_mockhsm build")
}
