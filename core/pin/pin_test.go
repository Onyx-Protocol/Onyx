package pin

import (
	"context"
	"testing"
	"time"

	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestWaitForPin(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()

	p := newPin(dbtx, "test", 0)
	s := &Store{pins: map[string]*pin{"test": p}}

	sctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	ch := make(chan error)
	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			ch <- ctx.Err()
		case <-s.WaitForPin("test", 1):
			ch <- nil
		}
	}(sctx)

	err := p.raiseTo(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	err = <-ch
	if err != nil {
		t.Fatal(err)
	}
}

func TestComplete(t *testing.T) {
	dbURL, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()

	store := NewStore(db)
	err := store.CreatePin(ctx, "x", 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	go store.ListenQueue(ctx, "x", dbURL)

	err = store.addToQueue(ctx, "x", 2)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	height, err := store.HeightToProcess(ctx, "testprocess", "x")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = store.Complete(ctx, "testprocess", "x", height)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	p := <-store.pin("x")
	if p.getHeight() != 2 {
		t.Errorf("got height = %d want %d", p.getHeight(), 2)
	}
}

func TestRelease(t *testing.T) {
	dbURL, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()

	store := NewStore(db)
	err := store.CreatePin(ctx, "x", 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	go store.ListenQueue(ctx, "x", dbURL)

	err = store.addToQueue(ctx, "x", 2)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	height, err := store.HeightToProcess(ctx, "testprocess", "x")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	err = store.Release(ctx, "testprocess", "x", height)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	p := <-store.pin("x")
	if p.getHeight() != 1 {
		t.Errorf("got height = %d want %d", p.getHeight(), 1)
	}
}
