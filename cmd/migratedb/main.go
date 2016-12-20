// Command migratedb applies database migrations to the
// specified database.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"chain/core/migrate"
	_ "chain/database/pg"
	"chain/database/sql"
)

const help = `Usage: migratedb [-d url] [-status]

Command migratedb applies migrations to the specified database.
If the database URL is not provided, the local 'core' database is used.

Providing the -status flag will not run any migrations, but will
instead print out the status of each migration.
`

var (
	flagD      = flag.String("d", "postgres:///core?sslmode=disable", "database")
	flagStatus = flag.Bool("status", false, "print all migrations and their status")
	flagH      = flag.Bool("h", false, "show help")
)

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func main() {
	log.SetPrefix("migratedb: ")
	log.SetFlags(0)
	flag.Usage = func() { fmt.Println(help) }
	flag.Parse()
	if *flagH || *flagD == "" {
		flag.Usage()
		flag.PrintDefaults()
		return
	}

	// Create a database connection.
	db, err := sql.Open("hapg", *flagD)
	if err != nil {
		fatalf("unable to connect to %s: %v\n", *flagD, err)
	}
	defer db.Close()

	if *flagStatus {
		err = migrate.PrintStatus(db)
	} else {
		err = migrate.Run(db)
	}
	if err != nil {
		fatalf("error: %s\n", err)
	}
}
