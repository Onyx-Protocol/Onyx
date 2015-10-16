package pg

import (
	"database/sql/driver"
	"fmt"

	"github.com/lib/pq"
)

// SchemaDriver is a postgres driver that
// sets the search_path to a specific schema.
type SchemaDriver string

// Open satisfies the Driver interface defined in db/sql
func (d SchemaDriver) Open(name string) (driver.Conn, error) {
	conn, err := pq.Open(name)
	if err != nil {
		return nil, err
	}

	execer := conn.(driver.Execer)
	sp := fmt.Sprintf("SET search_path TO %s, public, pg_catalog", pq.QuoteIdentifier(string(d)))
	_, err = execer.Exec(sp, []driver.Value{})
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// IsUniqueViolation returns true if the given error is a Postgres unique
// constraint violation error.
func IsUniqueViolation(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code.Name() == "unique_violation"
}

// IsForeignKeyViolation returns true if the given error is a Postgres
// foreign-key constraint violation error.
func IsForeignKeyViolation(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code.Name() == "foreign_key_violation"
}
