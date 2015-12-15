package pgtest

import (
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
)

// NewContext begins a transaction in the connected database,
// executes the given SQL statements inside the transaction,
// and returns a new Context containing the in-progress transaction.
// The transaction also has a Begin method that returns itself,
// so it can be used in functions that expect to begin their own
// new transaction.
func NewContext(tb testing.TB, sql ...string) context.Context {
	ctx := context.Background()
	dbtx := txWithSQL(ctx, tb, sql...)
	return pg.NewContext(ctx, dbtx)
}

// Finish calls Rollback on the dbtx stored in ctx, if any.
func Finish(ctx context.Context) {
	if dbtx, ok := pg.FromContext(ctx).(pg.Tx); ok {
		dbtx.Rollback(ctx)
	}
}
