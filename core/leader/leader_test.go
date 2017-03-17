package leader

import (
	"context"
	"sync"
	"testing"

	"chain/database/pg/pgtest"
)

func TestFailover(t *testing.T) {
	var l1, l2 *Leader
	var wg1, wg2 sync.WaitGroup
	wg1.Add(1)
	wg2.Add(1)

	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	ctx1, cancel1 := context.WithCancel(ctx)
	ctx2, cancel2 := context.WithCancel(ctx)
	defer cancel1()
	defer cancel2()

	// Start up the first leader process. It should immediately become
	// leader.
	l1 = Run(ctx1, db, ":1999", func(_ context.Context) {
		// Since the lead func hasn't completed yet, the leader
		// state should still be 'recovering'.
		if s := l1.State(); s != Recovering {
			t.Errorf("for first process state, got %s want %s", s, Recovering)
		}

		t.Log("first process is now leader")
		wg1.Done()
	})

	// Wait for the first process lead func to complete. The first process
	// should be then be 'leading'.
	wg1.Wait()
	if s := l1.State(); s != Leading {
		t.Errorf("the first process state, got %s want %s", s, Leading)
	}
	addr, err := l1.Address(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if addr != l1.address {
		t.Errorf("leader Address() got %s, want %s", addr, l1.address)
	}

	// Start up the second leader process. It should be following.
	l2 = Run(ctx2, db, ":2000", func(_ context.Context) {
		// When this process takes over leadership, it should be
		// 'recovering'.
		if s := l2.State(); s != Recovering {
			t.Errorf("for second process state, got %s want %s", s, Recovering)
		}

		t.Log("second process is now leader")
		wg2.Done()
	})
	if s := l2.State(); s != Following {
		t.Errorf("for second process state, got %s want %s", s, Following)
	}

	// Kill the first process by cancelling its context. Then wait for the
	// second process to take over leadership.
	cancel1()
	wg2.Wait()
	if s := l1.State(); s != Leading {
		t.Errorf("the second process state, got %s want %s", s, Leading)
	}
	addr, err = l2.Address(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if addr != l2.address {
		t.Errorf("leader Address() got %s, want %s", addr, l2.address)
	}
}
