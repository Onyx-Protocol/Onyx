package pg

import (
	"database/sql/driver"
	"fmt"
	"net"
	"net/url"

	"github.com/lib/pq"

	chainnet "chain/net"
)

// SchemaDriver is a postgres driver that
// sets the search_path to a specific schema.
type SchemaDriver string

// Open satisfies the Driver interface defined in db/sql
func (d SchemaDriver) Open(name string) (driver.Conn, error) {
	name, err := resolveURI(name)
	if err != nil {
		return nil, err
	}

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

func resolveURI(rawURI string) (string, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return "", err
	}

	if u.Host == "" {
		// postgres specifies localhost with the empty string
		return rawURI, nil
	}

	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		// If there's an error, it might be because there's no
		// port on uri.Host, which is totally fine. If there's
		// another problem, it will get caught later anyway, so
		// carry on!
		host = u.Host
	}

	addrs, err := chainnet.LookupHost(host)
	if err != nil {
		return "", err
	}

	addr := addrs[0] // there should only be one address

	if port != "" {
		addr = net.JoinHostPort(addr, port)
	}

	u.Host = addr
	return u.String(), nil
}
