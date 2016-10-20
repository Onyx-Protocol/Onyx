// Command migratedb applies database migrations to the
// specified database.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "chain/database/pg"
	"chain/database/sql"
)

const help = `
Usage:

	migratedb [-d url] [-dryrun] [-status] [migration]

Command migratedb applies migrations to the specified
database.

If the database URL is not provided, the local 'core'
database is used.

Providing the -status flag will not run any migrations, but will
instead print out the status of each migration.
`

var (
	flagD      = flag.String("d", "postgres:///core?sslmode=disable", "database")
	flagStatus = flag.Bool("status", false, "print all migrations and their status")
	flagDry    = flag.Bool("dryrun", false, "print but don't execute migrations")
	flagH      = flag.Bool("h", false, "show help")

	dbURL string
)

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func main() {
	log.SetPrefix("migratedb: ")
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-d url] [-dryrun] [-status] [migration]\n", os.Args[0])
	}
	flag.Parse()
	args := flag.Args()
	if *flagH || *flagD == "" || (*flagStatus && len(args) > 0) {
		fmt.Println(strings.TrimSpace(help))
		fmt.Print("\nFlags:\n\n")
		flag.PrintDefaults()
		return
	}

	// Determine the directory with migrations using the $CHAIN environment
	// variable if it's available.
	migrationsDir := "migrations"
	if chain := os.Getenv("CHAIN"); chain != "" {
		migrationsDir = filepath.Join(chain, "migrations")
	}

	// Create a database connection.
	db, err := sql.Open("hapg", *flagD)
	if err != nil {
		fatalf("unable to connect to %s: %v\n", *flagD, err)
	}
	defer db.Close()

	// Retrieve the current state of migrations.
	migrations, err := loadMigrations(db, migrationsDir)
	if err != nil {
		fatalf("unable to load current state: %s\n", err.Error())
	}

	// If -status is set, just print all known migrations and their current status,
	// then exit.
	if *flagStatus {
		fmt.Printf("%-60s\t%-6s\t%s\n", "filename", "hash", "applied_at")
		for _, m := range migrations {
			appliedAt := "(pending)"
			if m.AppliedAt != nil {
				appliedAt = m.AppliedAt.Format(time.RFC3339)
			}
			fmt.Printf("%-60s\t%-6s\t%s\n", m.Filename, m.Hash[:6], appliedAt)
		}
		return
	}

	var file string
	if len(args) > 0 {
		file = args[0]
	}

	var (
		found           bool
		migrationsToRun []migration
	)
	for _, m := range migrations {
		// Keep track of all of the migrations that need to be run.
		if !m.Applied {
			migrationsToRun = append(migrationsToRun, m)
		}

		if file == m.Filename {
			found = true
			break
		}
	}

	if file != "" && !found {
		fatalf("unable to find migration: %s\n", file)
	}
	if file != "" && (len(migrationsToRun) == 0 || file != migrationsToRun[len(migrationsToRun)-1].Filename) {
		fatalf("migration already applied: %s\n", file)
	}

	for _, m := range migrationsToRun {
		fmt.Println("Pending migration:", m.Filename)
		if !*flagDry {
			err := runMigration(db, *flagD, migrationsDir, m)
			if err != nil {
				fatalf("unable to run migration %s: %v\n", m.Filename, err)
			}
			fmt.Printf("Successfully ran %s migration on %s\n", m.Filename, *flagD)
		}
	}
}
