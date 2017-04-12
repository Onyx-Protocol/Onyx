//+build no_mockhsm

package main

import "chain/database/sql"

func createBlockKeyPair(db *sql.DB, args []string) {
	fatalln("error: create-block-keypair disabled in prod build")
}
