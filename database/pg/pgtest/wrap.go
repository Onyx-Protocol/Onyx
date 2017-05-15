package pgtest

import (
	"database/sql"
	"database/sql/driver"
	"runtime"
	"testing"

	"github.com/lib/pq"
)

// WrapDB opens a new connection to the database at the provided URL,
// but with a driver that calls wrapFn on every driver.Stmt.Exec call
// and driver.Stmt.Query call.
//
// It also registers a finalizer for the DB, so callers can discard
// it without closing it explicitly.
func WrapDB(t testing.TB, url string, wrapFn func(string)) *sql.DB {
	// Register a new SQL driver that will wrapFn on every driver.Stmt
	// Exec and Query call.
	driverName := pickName("wrappeddriver")
	sql.Register(driverName, &wrappedDriver{fn: wrapFn})
	db, err := sql.Open(driverName, url)
	if err != nil {
		t.Fatal(err)
	}
	runtime.SetFinalizer(db, (*sql.DB).Close)
	return db
}

type wrappedDriver struct {
	fn func(string)
}

func (d *wrappedDriver) Open(name string) (driver.Conn, error) {
	conn, err := pq.Open(name)
	if err != nil {
		return conn, err
	}
	return wrappedConn{fn: d.fn, backing: conn}, nil
}

type wrappedConn struct {
	fn      func(string)
	backing driver.Conn
}

func (c wrappedConn) Prepare(query string) (driver.Stmt, error) {
	c.fn(query)
	stmt, err := c.backing.Prepare(query)
	if err != nil {
		return stmt, err
	}
	return stmt, nil
}

func (c wrappedConn) Close() error {
	return c.backing.Close()
}

func (c wrappedConn) Begin() (driver.Tx, error) {
	return c.backing.Begin()
}
