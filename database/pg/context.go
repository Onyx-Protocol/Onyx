package pg

import (
	"database/sql"
	"fmt"

	"golang.org/x/net/context"
)

// DB holds methods common to the DB, Tx, and Stmt types
// in package sql.
type DB interface {
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
	Exec(string, ...interface{}) (sql.Result, error)
}

// Committer provides methods to commit or roll back a single transaction.
type Committer interface {
	Commit() error
	Rollback() error
}

// Tx represents a SQL transaction.
// Type sql.Tx satisfies this interface.
type Tx interface {
	DB
	Committer
}

// Beginner is used by Begin to create a new transaction.
// It is an optional alternative to the Begin signature provided by
// package sql.
type Beginner interface {
	Begin() (Tx, error)
}

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// dbKey is the key for DB values in Contexts.  It is
// unexported; clients use pg.NewContext and pg.FromContext
// instead of using this key directly.
var dbKey key

var logQueries bool

// EnableQueryLogging enables or disables log output for queries. For
// simplicity, it is not thread-safe. It makes the most sense to set this once
// at process boot.
func EnableQueryLogging(e bool) {
	logQueries = e
}

// Begin opens a new transaction on the database
// stored in ctx. The stored database must
// provide a Begin method like sql.DB or satisfy
// the interface Beginner.
// Begin returns the new transaction and
// a new context with the transaction as its
// associated database.
func Begin(ctx context.Context) (Committer, context.Context, error) {
	tx, err := begin(FromContext(ctx))
	if err != nil {
		return nil, nil, err
	}
	ctx = NewContext(ctx, tx)
	return tx, ctx, nil
}

func begin(db DB) (Tx, error) {
	type beginner interface {
		Begin() (*sql.Tx, error)
	}
	switch d := db.(type) {
	case beginner: // e.g. *sql.DB
		return d.Begin()
	case Beginner: // e.g. pgtest.noCommitDB
		return d.Begin()
	}
	return nil, fmt.Errorf("unknown db type %T", db)
}

// NewContext returns a new Context that carries value db.
func NewContext(ctx context.Context, db DB) context.Context {
	return context.WithValue(ctx, dbKey, db)
}

// FromContext returns the DB value stored in ctx.
// If there is no DB value, FromContext panics.
func FromContext(ctx context.Context) DB {
	db := ctx.Value(dbKey).(DB)
	if !logQueries {
		return db
	}

	if ldb, ok := db.(*Logger); ok {
		return ldb
	}
	return &Logger{db, ctx}
}
