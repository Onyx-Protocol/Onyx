package pg

import (
	"context"

	"chain/database/sql"
)

// DB holds methods common to the DB, Tx, and Stmt types
// in package sql.
type DB interface {
	Query(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRow(context.Context, string, ...interface{}) *sql.Row
	Exec(context.Context, string, ...interface{}) (sql.Result, error)
}

// Committer provides methods to commit or roll back a single transaction.
type Committer interface {
	Commit(context.Context) error
	Rollback(context.Context) error
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
	Begin(context.Context) (Tx, error)
}

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// dbKey is the key for DB values in Contexts.  It is
// unexported; clients use pg.NewContext and pg.FromContext
// instead of using this key directly.
var dbKey key

// NewContext returns a new Context that carries value db.
func NewContext(ctx context.Context, db DB) context.Context {
	return context.WithValue(ctx, dbKey, db)
}

// FromContext returns the DB value stored in ctx.
// If there is no DB value, FromContext panics.
func FromContext(ctx context.Context) DB {
	return ctx.Value(dbKey).(DB)
}
