package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"chain/errors"
)

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
func (m byFilename) Less(i, j int) bool { return m[i].Filename < m[j].Filename }
func (m byFilename) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }

// migration describes a single migration.
type migration struct {
	Filename  string
	Hash      string
	AppliedAt *time.Time
	Applied   bool
}

func (m migration) String() string {
	return fmt.Sprintf("%s - %s", m.Filename, m.Hash[:5])
}

// loadMigrations returns a slice of all the defined migrations.
func loadMigrations(db *sql.DB, migrationsDir string) ([]migration, error) {
	// Create the migrations table if not yet created.
	if _, err := db.Exec(createMigrationTableSQL); err != nil {
		return nil, errors.Wrapf(err, "creating migration table")
	}

	files, err := migrationFiles(migrationsDir)
	if err != nil {
		return nil, errors.Wrapf(err, "loading migrations from directory")
	}

	applied, err := appliedMigrations(db)
	if err != nil {
		return nil, errors.Wrapf(err, "loading applied migrations from database")
	}

	h := sha256.New()
	migrations := make([]migration, 0, len(files))
	for _, filename := range files {
		b, err := ioutil.ReadFile(filepath.Join(migrationsDir, filename))
		if err != nil {
			return nil, errors.Wrapf(err, "reading migration %s", filename)
		}

		h.Reset()
		h.Write(b)
		m := migration{
			Filename: filename,
			Hash:     hex.EncodeToString(h.Sum(nil)),
		}

		// Pull in data from the database if the migration was applied.
		if appliedMig, ok := applied[m.Filename]; ok {
			if appliedMig.Hash != m.Hash {
				return nil, fmt.Errorf("%s: applied hash %s doesn't match %s",
					m.Filename, appliedMig.Hash, m.Hash)
			}
			m.AppliedAt = appliedMig.AppliedAt
			m.Applied = true
		}

		migrations = append(migrations, m)
	}
	sort.Sort(byFilename(migrations))
	return migrations, nil
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

// appliedMigrations returns a map of  all migrations that have already
// been applied. The keys to the map are the migration filenames.
func appliedMigrations(db *sql.DB) (map[string]migration, error) {
	const q = `SELECT filename, hash, applied_at FROM migrations`

	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	migs := map[string]migration{}
	for rows.Next() {
		var mig migration
		err := rows.Scan(&mig.Filename, &mig.Hash, &mig.AppliedAt)
		if err != nil {
			return nil, err
		}
		migs[mig.Filename] = mig
	}
	return migs, rows.Err()
}

func runMigration(db *sql.DB, dbURL, migrationDir string, m migration) error {
	if filepath.Ext(m.Filename) == ".go" {
		return runGoMigration(db, dbURL, migrationDir, m)
	} else if filepath.Ext(m.Filename) == ".sql" {
		return runSQLMigration(db, migrationDir, m)
	}
	return errors.New("invalid migration filetype")
}

func runGoMigration(db *sql.DB, dbURL, migrationDir string, m migration) error {
	cmd := exec.Command("go", "run", filepath.Join(migrationDir, m.Filename))
	cmd.Env = append(os.Environ(), "DB_URL="+dbURL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(out))
	}

	return insertAppliedMigration(db, m)
}

func runSQLMigration(db *sql.DB, migrationDir string, m migration) error {
	migration, err := ioutil.ReadFile(filepath.Join(migrationDir, m.Filename))
	if err != nil {
		return err
	}

	_, err = db.Exec(string(migration))
	if err != nil {
		return err
	}

	// The migration and the insertion cannot be grouped in a single
	// transaction, because some migrations contain SQL that cannot be
	// run within a transaction.
	return insertAppliedMigration(db, m)
}

func insertAppliedMigration(db *sql.DB, m migration) error {
	const q = `
		INSERT INTO migrations (filename, hash, applied_at)
		VALUES($1, $2, NOW())
	`
	_, err := db.Exec(q, m.Filename, m.Hash)
	if err != nil {
		return errors.Wrap(err, "recording applied migration")
	}
	return nil
}
