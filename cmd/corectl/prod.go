//+build prod

package main

import (
	"fmt"

	"chain/database/sql"
)

func reset(*sql.DB, []string) {
	fatalln("error: reset disabled in prod build")
}

func versionProdPrintln() {
	fmt.Println("production: true")
}
