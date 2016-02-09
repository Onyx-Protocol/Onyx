// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot open db: %v\n", err)
		os.Exit(1)
	}

	_, err = db.Exec("SELECT 1")
	if err != nil {
		fmt.Fprintf(os.Stderr, "running select: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("migrated")
}
