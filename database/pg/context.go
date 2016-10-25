package pg

import (
	"context"
	"fmt"

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

// Begin opens a new transaction on the database
// stored in ctx. The stored database must
// provide a Begin method like sql.DB or satisfy
// the interface Beginner.
// Begin returns the new transaction and
// a new context with the transaction as its
// associated database.
//
// Note: if a transaction is already pending in the passed-in context,
// this function does not create a new one but returns the existing
// one.
func Begin(ctx context.Context) (Committer, context.Context, error) {
	db := FromContext(ctx)

	if dbtx, ok := db.(Tx); ok {
		return newNestedTx(ctx, dbtx)
	}

	dbtx, err := begin(db, ctx)
	if err != nil {
		return nil, nil, err
	}
	ctx = NewContext(ctx, dbtx)
	return dbtx, ctx, nil
}

func begin(db DB, ctx context.Context) (Tx, error) {
	type beginner interface {
		Begin(context.Context) (*sql.Tx, error)
	}
	switch d := db.(type) {
	case beginner: // e.g. *sql.DB
		return d.Begin(ctx)
	case Beginner: // e.g. pgtest.noCommitDB
		return d.Begin(ctx)
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
	return ctx.Value(dbKey).(DB)
}
