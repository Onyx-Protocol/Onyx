package pg

import (
	"database/sql"

	"golang.org/x/net/context"
)

// DB holds methods common to the DB, Tx, and Stmt types
// in package sql.
type DB interface {
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
	Exec(string, ...interface{}) (sql.Result, error)
}

type Tx interface {
	Commit() error
	Rollback() error
}

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// dbKey is the key for DB values in Contexts.  It is
// unexported; clients use pg.NewContext and pg.FromContext
// instead of using this key directly.
var dbKey key = 0

// Begin opens a new transaction on the database
// stored in ctx. It returns the transaction and
// a new context with the transaction as its
// associated database.
func Begin(ctx context.Context) (Tx, context.Context, error) {
	type beginner interface {
		Begin() (*sql.Tx, error)
	}
	db := FromContext(ctx).(beginner)
	tx, err := db.Begin()
	if err != nil {
		return nil, nil, err
	}
	ctx = NewContext(ctx, tx)
	return tx, ctx, nil
}

// NewContext returns a new Context that carries value db.
func NewContext(ctx context.Context, db DB) context.Context {
	return context.WithValue(ctx, dbKey, db)
}

// FromContext returns the DB value stored in ctx.
// If there is no DB value, FromContext panics.
func FromContext(ctx context.Context) DB {
	return ctx.Value(dbKey).(DB)
}
