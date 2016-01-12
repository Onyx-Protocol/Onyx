// migratedb applies the specified migration
// to the specified database or target
package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	_ "github.com/lib/pq"
)

const help = `
Usage:

	migratedb [-t target] [-d url] [migration]

Command migratedb applies migrations to the specified
database or target.

Either the database or the target flag must be specified,
but not both.
`

var (
	flagD = flag.String("d", "", "database")
	flagT = flag.String("t", "", "target")
	flagH = flag.Bool("h", false, "show help")

	dbURL string
)

func main() {
	log.SetPrefix("appenv: ")
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-t target] [-d url] [migration]\n", os.Args[0])
	}
	flag.Parse()
	args := flag.Args()
	if *flagH || (*flagT == "") == (*flagD == "") || len(args) == 0 {
		fmt.Println(strings.TrimSpace(help))
		fmt.Print("\nFlags:\n\n")
		flag.PrintDefaults()
		return
	}

	if *flagD != "" {
		dbURL = *flagD
	}
	if *flagT != "" {
		var err error
		dbURL, err = getTargetDBURL(*flagT)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to get target DB_URL: %v\n", err)
			os.Exit(1)
		}
	}

	file := args[0]
	err := runMigration(dbURL, file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to run migration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("successfully ran migration %s on %s\n", file, *flagT+*flagD)
}

func getTargetDBURL(target string) (string, error) {
	out, err := exec.Command("appenv", "-t", target, "DB_URL").CombinedOutput()
	if err != nil {
		return "", errors.New(string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func runMigration(dbURL, file string) error {
	if strings.HasSuffix(file, ".go") {
		return runGoMigration(dbURL, file)
	} else if strings.HasSuffix(file, ".sql") {
		return runSQLMigration(dbURL, file)
	}
	return errors.New("invalid migration filetype")
}

func runGoMigration(dbURL, file string) error {
	cmd := exec.Command("go", "run", file)
	cmd.Env = append(os.Environ(), "DB_URL="+dbURL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(out))
	}
	return nil
}

func runSQLMigration(dbURL, file string) error {
	migration, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(string(migration))
	if err != nil {
		return err
	}

	return tx.Commit()
}
