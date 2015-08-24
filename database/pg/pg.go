package pg

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io/ioutil"

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

// LoadFile runs all the queries in a file on a database connection
func LoadFile(db *sql.DB, filepath string) error {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(file))
	return err
}
