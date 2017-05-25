// Package pg provides small utilities for the lib/pq
// database driver.
//
// It also registers the sql.Driver "hapg", which can
// resolve uris from the high-availability postgres package.
package pg

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/lib/pq"

	chainnet "chain/net"
)

// DB holds methods common to the DB, Tx, and Stmt types
// in package sql.
type DB interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

// TODO: move this under chain/hapg
type hapgDriver struct{}

func NewDriver() driver.Driver {
	return hapgDriver{}
}

func (d hapgDriver) Open(name string) (driver.Conn, error) {
	name, err := resolveURI(name)
	if err != nil {
		return nil, err
	}

	return pq.Open(name)
}

func init() {
	sql.Register("hapg", hapgDriver{})
}

// IsUniqueViolation returns true if the given error is a Postgres unique
// constraint violation error.
func IsUniqueViolation(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code.Name() == "unique_violation"
}

// IsValidJSONB returns true if the provided bytes may be stored
// in a Postgres JSONB data type. It validates that b is valid
// utf-8 and valid json. It also verifies that it does not include
// the \u0000 escape sequence, unsupported by the jsonb data type:
// https://www.postgresql.org/message-id/E1YHHV8-00032A-Em@gemulon.postgresql.org
func IsValidJSONB(b []byte) bool {
	var v interface{}
	err := json.Unmarshal(b, &v)
	return err == nil && utf8.Valid(b) && !containsNullByte(v)
}

func containsNullByte(v interface{}) (found bool) {
	const nullByte = '\u0000'
	switch t := v.(type) {
	case bool:
		return false
	case float64:
		return false
	case string:
		return strings.ContainsRune(t, nullByte)
	case []interface{}:
		for _, v := range t {
			found = found || containsNullByte(v)
		}
		return found
	case map[string]interface{}:
		for k, v := range t {
			found = found || containsNullByte(k) || containsNullByte(v)
		}
		return found
	case nil:
		return false
	default:
		panic(fmt.Errorf("unknown json type %T", v))
	}
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
