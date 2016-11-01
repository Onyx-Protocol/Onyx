package pin

import (
	"context"
	"testing"
	"time"

	"chain/database/pg/pgtest"
	"chain/database/sql"
)

func TestWaitForHeight(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()

	p := newPin(dbtx, "test", 0)

	sctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	ch := make(chan error)
	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			ch <- ctx.Err()
		case <-p.WaitForHeight(1):
			ch <- nil
		}
	}(sctx)

	err := p.RaiseTo(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	err = <-ch
	if err != nil {
		t.Fatal(err)
	}
}

func TestListen(t *testing.T) {
	ctx := context.Background()

	dbURL := pgtest.DBURL
	if dbURL == "" {
		dbURL = pgtest.DefaultURL
	}

	db, err := sql.Open("postgres", dbURL)

	p := newPin(db, "test", 0)

	sctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	ch := make(chan error)

	go p.Listen(sctx, dbURL)
	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			ch <- ctx.Err()
		case <-p.WaitForHeight(1):
			ch <- nil
		}
	}(sctx)

	_, err = db.Exec(ctx, `SELECT pg_notify('pin-test'::text, '1')`)
	if err != nil {
		t.Fatal(err)
	}

	err = <-ch
	if err != nil {
		t.Fatal(err)
	}
}
