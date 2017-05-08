package sqlutil

import (
	"context"
	"database/sql/driver"
	"fmt"

	"chain/log"
)

// TODO(kr): many databases—Postgres in particular—report the
// execution time of each query or statement as measured on the
// database backend. Find a way to record that timing info in
// the trace.

const maxArgsLogLen = 20 // bytes

func logQuery(ctx context.Context, query string, args interface{}) {
	s := fmt.Sprint(args)
	if len(s) > maxArgsLogLen {
		s = s[:maxArgsLogLen-3] + "..."
	}
	log.Printkv(ctx, "query", query, "args", s)
}

type logDriver struct {
	driver driver.Driver
}

// LogDriver returns a Driver that logs each query
// before forwarding it to d.
func LogDriver(d driver.Driver) driver.Driver {
	return &logDriver{d}
}

func (ld *logDriver) Open(name string) (driver.Conn, error) {
	c, err := ld.driver.Open(name)
	return &logConn{c}, err
}

type logConn struct {
	driver.Conn
}

func (lc *logConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := lc.Conn.Prepare(query)
	return &logStmt{query, stmt}, err
}

func (lc *logConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	execer, ok := lc.Conn.(driver.Execer)
	if !ok {
		return nil, driver.ErrSkip
	}
	logQuery(context.Background(), query, args)
	return execer.Exec(query, args)
}

func (lc *logConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	queryer, ok := lc.Conn.(driver.Queryer)
	if !ok {
		return nil, driver.ErrSkip
	}
	logQuery(context.Background(), query, args)
	return queryer.Query(query, args)
}

// TODO(kr): implement context variants
// (but don't bother until lib/pq does first).
//func (lc *logConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error)
//func (lc *logConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error)
//func (lc *logConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error)

type logStmt struct {
	query string
	driver.Stmt
}

func (ls *logStmt) Exec(args []driver.Value) (driver.Result, error) {
	logQuery(context.Background(), ls.query, args)
	return ls.Stmt.Exec(args)
}

func (ls *logStmt) Query(args []driver.Value) (driver.Rows, error) {
	logQuery(context.Background(), ls.query, args)
	return ls.Stmt.Query(args)
}
