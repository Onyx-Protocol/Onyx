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
