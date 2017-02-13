//+build prod

package main

import "chain/database/sql"

func reset(db *sql.DB, args []string) {
	fatalln("error: reset disabled in prod build")
}
