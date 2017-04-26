package core

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/generator"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/core/txdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest"
)

// TestRecovery tests end-to-end blockchain recovery from an exit
// during landing a block.
func TestRecovery(t *testing.T) {
	if os.Getenv("LONG") == "" {
		t.Skip("skipping test; $LONG not set")
	}

	// Setup the test environment using a clean db.
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	dbURL, db := pgtest.NewDB(t, pgtest.SchemaPath)
	store := txdb.NewStore(db)
	c := prottest.NewChain(t, prottest.WithStore(store))
	g := generator.New(c, nil, db)
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	indexer := query.NewIndexer(db, c, pinStore)
	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go assets.ProcessBlocks(ctx)
	go accounts.ProcessBlocks(ctx)
	go indexer.ProcessBlocks(ctx)

	// Setup the transaction query indexer to index every transaction.
	indexer.RegisterAnnotator(accounts.AnnotateTxs)
	indexer.RegisterAnnotator(assets.AnnotateTxs)

	var err error
	pinHeight := c.Height()
	if pinHeight > 0 {
		pinHeight = pinHeight - 1
	}
	err = pinStore.CreatePin(ctx, account.PinName, pinHeight)
	if err != nil {
		t.Fatal(err)
	}
	err = pinStore.CreatePin(ctx, account.ExpirePinName, pinHeight)
	if err != nil {
		t.Fatal(err)
	}
	err = pinStore.CreatePin(ctx, account.DeleteSpentsPinName, pinHeight)
	if err != nil {
		t.Fatal(err)
	}
	err = pinStore.CreatePin(ctx, asset.PinName, pinHeight)
	if err != nil {
		t.Fatal(err)
	}
	err = pinStore.CreatePin(ctx, query.TxPinName, pinHeight)
	if err != nil {
		t.Fatal(err)
	}

	// Create two assets (USD & apples) and two accounts (Alice & Bob).
	var (
		usdTags = map[string]interface{}{"currency": "usd"}
		usd     = coretest.CreateAsset(ctx, t, assets, nil, "usd", usdTags)
		apple   = coretest.CreateAsset(ctx, t, assets, nil, "apple", nil)
		alice   = coretest.CreateAccount(ctx, t, accounts, "alice", nil)
		bob     = coretest.CreateAccount(ctx, t, accounts, "bob", nil)
	)
	// Issue some apples to Alice and a dollar to Bob.
	coretest.IssueAssets(ctx, t, c, g, assets, accounts, apple, 10, alice)
	coretest.IssueAssets(ctx, t, c, g, assets, accounts, usd, 1, bob)
	setupBlock := prottest.MakeBlock(t, c, g.PendingTxs())
	<-pinStore.AllWaiter(setupBlock.Height)

	// Submit a transfer between Alice and Bob but don't publish it in a block.
	coretest.Transfer(ctx, t, c, g, []txbuilder.Action{
		accounts.NewControlAction(bc.AssetAmount{AssetId: &usd, Amount: 1}, alice, nil),
		accounts.NewControlAction(bc.AssetAmount{AssetId: &apple, Amount: 1}, bob, nil),
		accounts.NewSpendAction(bc.AssetAmount{AssetId: &usd, Amount: 1}, bob, nil, nil),
		accounts.NewSpendAction(bc.AssetAmount{AssetId: &apple, Amount: 1}, alice, nil, nil),
	})
	poolTxs := g.PendingTxs()

	cancel()
	t.Logf("Collected %d pending transactions", len(poolTxs))
	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Run the `generateBlock` function repeatedly, each time starting
	// from the state of the test database initialized above. Each time,
	// the SQL driver will force the goroutine to exit at a different
	// query. After each forced exit, run `generateBlock` again to recover.
	// Despite crashing at a different point on each run, the resulting
	// states should be identical.
	timestamp := time.Now()
	var databaseDumps []string
	for n := int64(1); ; n++ {
		ctx = context.Background()
		t.Logf("Beginning run; will crash on %d-th SQL query", n)
		// Create a new, fresh database using the schema file we created
		// above. The database will be identical.
		cloneURL, err := pgtest.CloneDB(ctx, dbURL)
		if err != nil {
			t.Fatal(err)
		}

		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Open a new handle to the same database but using a driver that
		// will call our anonymous func to simulate crashes.
		crashed := make(chan struct{})
		completed := make(chan struct{})
		var calls int64
		wrappedDB := pgtest.WrapDB(t, cloneURL, func(q string) {
			num := atomic.AddInt64(&calls, 1)
			if num == n {
				t.Logf("crashing on query %d: %s\n", num, q)
				cancel()         // cancel the context so other goroutines clean up too
				close(crashed)   // let main goroutine know
				runtime.Goexit() // crash this goroutine
			}
		})

		go func() {
			err = generateBlock(runCtx, t, wrappedDB, timestamp, poolTxs)
			if err != nil {
				t.Errorf("generateBlock returned err %s", err)
			}
			cancel()
			close(completed)
		}()

		// Wait for the goroutine to finish or get killed on the N-th query.
		select {
		case <-crashed:
			t.Log("goroutine crashed on query")
		case <-completed:
			t.Log("goroutine completed successfully")
		}

		if atomic.LoadInt64(&calls) < n {
			// calls never reached n, so the goroutine completed without
			// simulating a crash.
			databaseDumps = append(databaseDumps, pgtest.Dump(t, cloneURL, false, "*_id_seq", "block_processors"))
			break
		}

		// We crashed at some point during block generation. Do it again,
		// without crashing.
		err = generateBlock(ctx, t, wrappedDB, timestamp, poolTxs)
		if err != nil {
			t.Fatal(err)
		}
		databaseDumps = append(databaseDumps, pgtest.Dump(t, cloneURL, false, "*_id_seq", "block_processors"))
	}

	if len(databaseDumps) < 2 {
		t.Fatal("no database dumps; did the wrapped driver get used?")
	}
	// Compare all of the pg_dumps. They should all be equal. Only
	// print the hashes though so that the test output isn't overwhelming.
	var prevHash string
	dumps := make(map[string]string)
	for n, v := range databaseDumps {
		hasher := md5.New()
		hasher.Write([]byte(v))
		hash := hex.EncodeToString(hasher.Sum(nil))
		if n >= 1 && prevHash != hash {
			t.Errorf("previous run %d - %s; current run %d - %s", n, prevHash, n+1, hash)
		}
		dumps[hash] = v
		prevHash = hash
	}

	// If the test failed, save the database dumps to a test directory
	// so that they can be manually examined and diffed.
	if len(dumps) > 1 {
		dir, err := ioutil.TempDir("", "recovery-test")
		if err != nil {
			t.Fatal(err)
		}
		for hash, dump := range dumps {
			err = ioutil.WriteFile(filepath.Join(dir, hash), []byte(dump), os.ModePerm)
			if err != nil {
				t.Fatal(err)
			}
		}
		t.Logf("Wrote database dumps to %s", dir)
	}
}

func generateBlock(ctx context.Context, t testing.TB, db pg.DB, timestamp time.Time, poolTxs []*legacy.Tx) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	store := txdb.NewStore(db)
	b1, err := store.GetBlock(ctx, 1)
	if err != nil {
		return err
	}

	c, err := protocol.NewChain(ctx, b1.Hash(), store, nil)
	if err != nil {
		return errors.Wrap(err)
	}
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	indexer := query.NewIndexer(db, c, pinStore)
	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)

	// Setup the transaction query indexer to index every transaction.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	indexer.RegisterAnnotator(assets.AnnotateTxs)
	indexer.RegisterAnnotator(accounts.AnnotateTxs)
	err = pinStore.LoadAll(ctx)
	if err != nil {
		return err
	}
	go assets.ProcessBlocks(ctx)
	go accounts.ProcessBlocks(ctx)
	go indexer.ProcessBlocks(ctx)

	block, snapshot, err := c.Recover(ctx)
	if err != nil {
		return err
	}

	b, s, err := c.GenerateBlock(ctx, block, snapshot, timestamp, poolTxs)
	if err != nil {
		return err
	}
	if len(b.Transactions) == 0 {
		return nil
	}
	err = c.CommitAppliedBlock(ctx, b, s)
	if err != nil {
		return err
	}

	<-pinStore.AllWaiter(b.Height)
	return nil
}
