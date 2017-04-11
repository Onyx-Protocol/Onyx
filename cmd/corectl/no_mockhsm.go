//+build no_mockhsm

package main

import "chain/database/sql"

func init() {
	mockHSM = false
}

func createBlockKeyPair(*sql.DB, []string) {
	fatalln("error: create-block-keypair disabled in this build")
}
