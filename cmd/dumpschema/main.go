package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"chain/core/migrate"
	"chain/database/pg"
)

const temporaryDatabase = `tmp_db_for_dump_schema`

func main() {
	chainDir := os.Getenv("CHAIN")
	if chainDir == "" {
		fmt.Fprintf(os.Stderr, "environment variable '$CHAIN' is unset")
		os.Exit(1)
	}

	must(run("dropdb", "--if-exists", temporaryDatabase))
	must(run("createdb", temporaryDatabase))

	db, err := sql.Open("hapg", fmt.Sprintf("postgres:///%s?sslmode=disable", temporaryDatabase))
	must(err)
	must(migrate.Run(db))

	var buf bytes.Buffer
	pgdump := exec.Command("pg_dump", "-sOx", temporaryDatabase)
	pgdump.Stdout = &buf
	pgdump.Stderr = os.Stderr
	must(pgdump.Run())

	f, err := os.Create(filepath.Join(chainDir, "core", "schema.sql"))
	must(err)
	defer f.Close()

	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(line, "--") || strings.Contains(line, "COMMENT") {
			continue
		}
		_, err = f.WriteString(line + "\n")
		must(err)
	}

	const q = `SELECT filename, hash FROM migrations ORDER BY filename`
	must(pg.ForQueryRows(context.Background(), db, q, func(filename, hash string) error {
		const insertStmt = "insert into migrations (filename, hash) values ('%s', '%s');\n"
		_, err = f.WriteString(fmt.Sprintf(insertStmt, filename, hash))
		return err
	}))

	must(db.Close())
	must(run("dropdb", temporaryDatabase))
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
