// Package migrate implements database migration for Chain Core.
package migrate

import (
	"context"
	"fmt"
	"time"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
)

// Run runs all built-in migrations.
func Run(db pg.DB) error {
	ctx := context.Background()

	// Create the migrations table if not yet created.
	_, err := db.ExecContext(ctx, createMigrationTableSQL)
	if err != nil {
		return errors.Wrap(err, "creating migration table")
	}

	err = convertOldStatus(db)
	if err != nil {
		return err
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
		_, err := db.ExecContext(ctx, m.SQL)
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

		log.Printkv(ctx, "migration", m.Name, "status", "success")
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

func (m migration) String() string {
	return fmt.Sprintf("%s - %s", m.Name, m.Hash[:5])
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
	err := db.QueryRowContext(ctx, q).Scan(&n)
	if err != nil {
		log.Fatalkv(ctx, log.KeyError, err)
	}
	if n == 0 {
		return nil // no schema; nothing has been applied
	}

	rows, err := db.QueryContext(ctx, `SELECT filename, hash, applied_at FROM migrations`)
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

// Well this is funny. We are going to migrate our migrations.
// We squashed our migration history into a single migration
// (2016-10-17.0.core.schema-snapshot.sql), but some deployed
// systems are running a database from before the squash.
// So we'll detect that case and convert it here to the new
// format before attempting to run any new migrations.
// If they are running a schema older than the last migration
// just before the squash, we cannot help them here.
func convertOldStatus(db pg.DB) error {
	ctx := context.Background()

	// last migration in the old regime
	const q = `
		SELECT count(*) FROM migrations
		WHERE filename='2016-10-10.0.mockhsm.add-key-types.sql'
		AND hash='00ac8143767fe4a44855cab1ec57afd52c44fd4d727055db9e8584c3e9b10983'
	`
	var n int
	err := db.QueryRowContext(ctx, q).Scan(&n)
	if err != nil {
		return errors.Wrap(err)
	}
	if n == 0 {
		return nil // no conversion necessary/possible
	}

	_, err = db.ExecContext(ctx, `
		TRUNCATE migrations;

		INSERT INTO migrations (filename, hash, applied_at)
		VALUES (
			'2016-10-17.0.core.schema-snapshot.sql',
			'cff5210e2d6af410719c223a76443f73c5c12fe875f0efecb9a0a5937cf029cd',
			now()
		);
	`)
	return errors.Wrap(err)
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
	_, err := db.ExecContext(ctx, q, m.Name, m.Hash)
	if err != nil {
		return errors.Wrap(err, "recording applied migration")
	}
	return nil
}
