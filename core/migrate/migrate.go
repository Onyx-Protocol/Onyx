package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
)

// Run runs all built-in migrations.
func Run(db pg.DB) error {
	ctx := context.Background()

	// Create the migrations table if not yet created.
	_, err := db.Exec(ctx, createMigrationTableSQL)
	if err != nil {
		return errors.Wrap(err, "creating migration table")
	}

	err = loadStatus(db, migrations)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if !m.AppliedAt.IsZero() {
			continue
		}
		fmt.Println("Pending migration:", m.Name)
		_, err := db.Exec(ctx, m.SQL)
		if err != nil {
			return errors.Wrapf(err, "migration %s", m.Name)
		}

		// The migration and the insertion cannot be grouped in a single
		// transaction, because some migrations contain SQL that cannot be
		// run within a transaction.
		err = insertAppliedMigration(db, m)
		if err != nil {
			return err
		}

		log.Write(ctx, "migration", m.Name, "status", "success")
	}
	return nil
}

// PrintStatus prints the status of each built-in migration.
func PrintStatus(db pg.DB) error {
	err := loadStatus(db, migrations)
	if err != nil {
		return err
	}

	fmt.Printf("%-60s\t%-6s\t%s\n", "filename", "hash", "applied_at")
	for _, m := range migrations {
		appliedAt := "(pending)"
		if !m.AppliedAt.IsZero() {
			appliedAt = m.AppliedAt.Format(time.RFC3339)
		}
		fmt.Printf("%-60s\t%-6s\t%s\n", m.Name, m.Hash[:6], appliedAt)
	}
	return nil
}

const createMigrationTableSQL = `
	  CREATE TABLE IF NOT EXISTS migrations (
		  filename text NOT NULL,
		  hash text NOT NULL,
		  applied_at timestamp with time zone DEFAULT now() NOT NULL,
		  PRIMARY KEY(filename)
	  );
`

// byFilename implements sort.Interface allowing sorting by the
// filename of the migration. This should sort by date because of
// the migration file naming scheme.
type byFilename []migration

func (m byFilename) Len() int           { return len(m) }
func (m byFilename) Less(i, j int) bool { return m[i].Name < m[j].Name }
func (m byFilename) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }

func (m migration) String() string {
	return fmt.Sprintf("%s - %s", m.Name, m.Hash[:5])
}

// migrationFiles returns a slice of the filenames of all migrations in the
// given directory.
func migrationFiles(migrationDirectory string) ([]string, error) {
	var filenames []string
	err := filepath.Walk(migrationDirectory, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(name) != ".sql" && filepath.Ext(name) != ".go" {
			return nil
		}

		filenames = append(filenames, filepath.Base(name))
		return nil
	})
	return filenames, err
}

// loadStatus sets AppliedAt on each item of ms that appears
// in table "migrations" in db.
// It is an error for the stored hash to be different
// from the migration's computed hash.
func loadStatus(db pg.DB, ms []migration) error {
	ctx := context.Background()
	const q = `
		SELECT count(*) FROM pg_tables
		WHERE schemaname='public' AND tablename='migrations'
	`
	var n int
	err := db.QueryRow(ctx, q).Scan(&n)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}
	if n == 0 {
		return nil // no schema; nothing has been applied
	}

	rows, err := db.Query(ctx, `SELECT filename, hash, applied_at FROM migrations`)
	if err != nil {
		return errors.Wrap(err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, hash string
		var t time.Time
		err := rows.Scan(&name, &hash, &t)
		if err != nil {
			return errors.Wrap(err)
		}
		m := find(name, ms)
		if m != nil {
			if m.Hash != hash {
				return errors.Wrap(fmt.Errorf("%s hash mismatch %s != %s", name, hash, m.Hash))
			}
			m.AppliedAt = t
		}
	}
	return errors.Wrap(rows.Err())
}

// find finds name in ms.
// It returns nil if not found.
func find(name string, ms []migration) *migration {
	for i := range ms {
		if ms[i].Name == name {
			return &ms[i]
		}
	}
	return nil
}

func insertAppliedMigration(db pg.DB, m migration) error {
	ctx := context.Background()

	const q = `
		INSERT INTO migrations (filename, hash, applied_at)
		VALUES($1, $2, NOW())
	`
	_, err := db.Exec(ctx, q, m.Name, m.Hash)
	if err != nil {
		return errors.Wrap(err, "recording applied migration")
	}
	return nil
}
