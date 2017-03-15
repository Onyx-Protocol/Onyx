//+build prod

package main

import (
	"fmt"

	"chain/database/sql"
)

func reset(db *sql.DB, args []string) {
	fatalln("error: reset disabled in prod build")
}

func createBlockKeyPair(db *sql.DB, args []string) {
	fatalln("error: create-block-keypair disabled in prod build")
}

func versionProdPrintln() {
	fmt.Println("production: true")
}
