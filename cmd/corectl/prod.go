//+build prod

package main

import "database/sql"

func reset(db *sql.DB, args []string) {
	fatalln("error: reset disabled in prod build")
}
