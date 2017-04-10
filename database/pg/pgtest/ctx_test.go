package pgtest

import (
	"context"
	"testing"
	"time"

	"chain/database/pg"
	"chain/errors"
)

func TestContextTimeout(t *testing.T) {
	ctx := context.Background()
	_, db := NewDB(t, SchemaPath)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	var err error
	for err == nil {
		err = pg.ForQueryRows(ctx, db, "SELECT 1", func(i int) {})
	}
	if errors.Root(err) != context.DeadlineExceeded {
		t.Fatalf("Got %s, want %s", err, context.DeadlineExceeded)
	}
}
