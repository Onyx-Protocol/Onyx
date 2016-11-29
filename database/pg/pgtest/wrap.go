package pgtest

import (
	"database/sql/driver"
	"runtime"
	"testing"

	"github.com/lib/pq"

	"chain-stealth/database/sql"
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
	// TODO(jackson): Ideally, we could call fn() here, but if fn() panics
	// the system will deadlock. We call fn() from Stmt.Exec / Stmt.Query
	// to avoid the deadlock.
	//
	// This will be fixed in Go 1.8:
	// https://go-review.googlesource.com/#/c/23576/
	stmt, err := c.backing.Prepare(query)
	if err != nil {
		return stmt, err
	}
	return wrappedStmt{fn: c.fn, query: query, backing: stmt}, nil
}

func (c wrappedConn) Close() error {
	return c.backing.Close()
}

func (c wrappedConn) Begin() (driver.Tx, error) {
	return c.backing.Begin()
}

type wrappedStmt struct {
	fn      func(string)
	query   string
	backing driver.Stmt
}

func (s wrappedStmt) NumInput() int {
	return s.backing.NumInput()
}

func (s wrappedStmt) Close() error {
	return s.backing.Close()
}

func (s wrappedStmt) Exec(args []driver.Value) (driver.Result, error) {
	s.fn(s.query)
	return s.backing.Exec(args)
}

func (s wrappedStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.fn(s.query)
	return s.backing.Query(args)
}
