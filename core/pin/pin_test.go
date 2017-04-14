package pin

import (
	"context"
	"testing"
	"time"

	"chain/database/pg/pgtest"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestCreatePin(t *testing.T) {
	db := pgtest.NewTx(t)
	ctx := context.Background()
	store := NewStore(db)

	err := store.CreatePin(ctx, "example", 100)
	if err != nil {
		t.Fatal(err)
	}
	if h := store.Height("example"); h != 100 {
		t.Errorf("pin height got %d, want %d", h, 100)
	}

	// Try to create the pin again but with a higher height.
	// The pin should be unchanged.
	err = store.CreatePin(ctx, "example", 20000)
	if err != nil {
		t.Fatal(err)
	}
	if h := store.Height("example"); h != 100 {
		t.Errorf("pin height got %d, want %d", h, 100)
	}
}

func TestListenPin(t *testing.T) {
	dbURL, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create two stores. One will update the pin, the other will listen to it.
	activeStore, passiveStore := NewStore(db), NewStore(db)
	err := activeStore.CreatePin(ctx, "example", 1)
	if err != nil {
		t.Fatal(err)
	}
	err = passiveStore.LoadAll(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Track updates to the pin height in the passive store.
	passiveStore.Listen(ctx, "example", dbURL)

	// Mark the pin as having completed block 2.
	pin := <-activeStore.pin("example")
	pin.complete(ctx, 2)

	// Wait for the passive store to recognize that block 2 has
	// been processed.
	<-passiveStore.PinWaiter("example", 2)
}

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

func TestProcessBlocks(t *testing.T) {
	db := pgtest.NewTx(t)
	store := NewStore(db)
	c := prottest.NewChain(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := store.CreatePin(ctx, "example", 0)
	if err != nil {
		t.Fatal(err)
	}

	var blockHeights []uint64
	go store.ProcessBlocks(ctx, c, "example", func(ctx context.Context, b *legacy.Block) error {
		// Wait for previous blocks to be processed.
		<-store.PinWaiter("example", b.Height-1)

		blockHeights = append(blockHeights, b.Height)
		return nil
	})

	// Make a handful of empty blocks, forcing the processor to process them.
	prottest.MakeBlock(t, c, nil)
	prottest.MakeBlock(t, c, nil)
	prottest.MakeBlock(t, c, nil)

	// Wait for the block processor to finish processing.
	<-store.AllWaiter(4)

	want := []uint64{1, 2, 3, 4}
	if !testutil.DeepEqual(blockHeights, want) {
		t.Errorf("processed block heights, got %#v want %#v", blockHeights, want)
	}
}
