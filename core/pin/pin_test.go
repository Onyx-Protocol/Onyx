package pin

import (
	"context"
	"testing"
	"time"

	"chain/database/pg/pgtest"
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
		case <-s.PinWaiter("test", 1):
			ch <- nil
		}
	}(sctx)

	err := p.complete(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	err = <-ch
	if err != nil {
		t.Fatal(err)
	}
}
