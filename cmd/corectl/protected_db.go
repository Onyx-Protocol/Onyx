//+build protected_db

package main

import "chain/database/sql"

func reset(*sql.DB, []string) {
	fatalln("error: reset disabled in prod build")
}
