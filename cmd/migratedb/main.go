// migratedb applies the specified migration
// to the specified database or target
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"chain/database/sql"
)

const help = `
Usage:

	migratedb [-t target] [-d url] [-dryrun] [-status] [migration]

Command migratedb applies migrations to the specified
database or target.

Either the database or the target flag must be specified,
but not both.

Providing the -status flag will not run any migrations, but will
instead print out the status of each migration.
`

var (
	flagD      = flag.String("d", "postgres:///core?sslmode=disable", "database")
	flagT      = flag.String("t", "", "target")
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
	log.SetPrefix("appenv: ")
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-t target] [-d url] [-dryrun] [-status] [migration]\n", os.Args[0])
	}
	flag.Parse()
	args := flag.Args()
	if *flagH || (*flagT == "") == (*flagD == "") || (*flagStatus && len(args) > 0) {
		fmt.Println(strings.TrimSpace(help))
		fmt.Print("\nFlags:\n\n")
		flag.PrintDefaults()
		return
	}

	if *flagD != "" {
		dbURL = *flagD
	}
	if *flagT != "" {
		if !*flagDry && !*flagStatus && isTargetRunning(*flagT) {
			fatalf("%s api is running; disable the app before running migrations.\n", *flagT)
		}

		var err error
		dbURL, err = getTargetDBURL(*flagT)
		if err != nil {
			fatalf("unable to get target DB_URL: %v\n", err)
		}
	}

	// Determine the directory with migrations using the $CHAIN environment
	// variable if it's available.
	migrationsDir := "migrations"
	if chain := os.Getenv("CHAIN"); chain != "" {
		migrationsDir = filepath.Join(chain, "migrations")
	}

	// Create a database connection.
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fatalf("unable to connect to %s: %v\n", dbURL, err)
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
			err := runMigration(db, dbURL, migrationsDir, m)
			if err != nil {
				fatalf("unable to run migration %s: %v\n", m.Filename, err)
			}
			fmt.Printf("Successfully ran %s migration on %s\n", m.Filename, *flagT+*flagD)
		}
	}
}

func getTargetDBURL(target string) (string, error) {
	out, err := exec.Command("appenv", "-t", target, "DB_URL").CombinedOutput()
	if err != nil {
		return "", errors.New(string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func isTargetRunning(t string) bool {
	c := http.Client{Timeout: 5 * time.Second}
	resp, err := c.Get(fmt.Sprintf("https://%s-api.chain.com/health", t))
	if err != nil {
		return false
	}
	return resp.StatusCode == http.StatusOK
}
