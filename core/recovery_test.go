package core

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"os"
	"runtime"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/core/txdb"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/mempool"
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
	dbURL, db := pgtest.NewDB(t, pgtest.SchemaPath)
	pool := mempool.New()
	store := txdb.NewStore(db)
	c := prottest.NewChainWithStorage(t, store, pool)
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	indexer := query.NewIndexer(db, c, pinStore)
	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go accounts.ProcessBlocks(ctx)

	// Setup the transaction query indexer to index every transaction.
	indexer.RegisterAnnotator(accounts.AnnotateTxs)
	indexer.RegisterAnnotator(assets.AnnotateTxs)

	// Create two assets (USD & apples) and two accounts (Alice & Bob).
	var (
		usdTags = map[string]interface{}{"currency": "usd"}
		usd     = coretest.CreateAsset(ctx, t, assets, nil, "usd", usdTags)
		apple   = coretest.CreateAsset(ctx, t, assets, nil, "apple", nil)
		alice   = coretest.CreateAccount(ctx, t, accounts, "alice", nil)
		bob     = coretest.CreateAccount(ctx, t, accounts, "bob", nil)
	)
	// Issue some apples to Alice and a dollar to Bob.
	_ = coretest.IssueAssets(ctx, t, c, assets, accounts, apple, 10, alice)
	_ = coretest.IssueAssets(ctx, t, c, assets, accounts, usd, 1, bob)

	prottest.MakeBlock(t, c)

	// Submit a transfer between Alice and Bob but don't publish it in a block.
	coretest.Transfer(ctx, t, c, []txbuilder.Action{
		accounts.NewControlAction(bc.AssetAmount{AssetID: usd, Amount: 1}, alice, nil),
		accounts.NewControlAction(bc.AssetAmount{AssetID: apple, Amount: 1}, bob, nil),
		accounts.NewSpendAction(bc.AssetAmount{AssetID: usd, Amount: 1}, bob, nil, nil),
		accounts.NewSpendAction(bc.AssetAmount{AssetID: apple, Amount: 1}, alice, nil, nil),
	})

	poolTxs, err := pool.Dump(ctx)
	if err != nil {
		t.Fatal(err)
	}

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
	for n := 1; ; n++ {
		// Create a new, fresh database using the schema file we created
		// above. The database will be identical.
		cloneURL, err := pgtest.CloneDB(ctx, dbURL)
		if err != nil {
			t.Fatal(err)
		}

		// Open a new handle to the same database but using a driver that
		// will call our anonymous func to simulate crashes.
		ch := make(chan error)
		calls := 0
		wrappedDB := pgtest.WrapDB(t, cloneURL, func(q string) {
			calls++

			if calls == n {
				t.Logf("crashing on query %d: %s\n", calls, q)
				close(ch)        // let main goroutine know
				runtime.Goexit() // crash this goroutine
			}
		})

		ctx := context.Background()
		go func() {
			err := generateBlock(ctx, t, wrappedDB, timestamp, poolTxs)
			ch <- err
		}()

		// Wait for the goroutine to finish or get killed on the N-th query.
		err, ok := <-ch
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			// The driver never crashed the goroutine, so n is now greater than
			// the total number of queries performed during `generateBlock`.
			databaseDumps = append(databaseDumps, pgtest.Dump(t, cloneURL, false, "*_id_seq"))
			break
		}

		// We crashed at some point during block generation. Do it again,
		// without crashing.
		err = generateBlock(ctx, t, wrappedDB, timestamp, poolTxs)
		if err != nil {
			t.Fatal(err)
		}
		databaseDumps = append(databaseDumps, pgtest.Dump(t, cloneURL, false, "*_id_seq"))
	}

	if len(databaseDumps) < 2 {
		t.Fatal("no database dumps; did the wrapped driver get used?")
	}
	// Compare all of the pg_dumps. They should all be equal. Only
	// print the hashes though so that the test output isn't overwhelming.
	var prevHash string
	for n, v := range databaseDumps {
		hasher := md5.New()
		hasher.Write([]byte(v))
		hash := hex.EncodeToString(hasher.Sum(nil))
		if n >= 1 && prevHash != hash {
			t.Errorf("previous run %d - %s; current run %d - %s", n, prevHash, n+1, hash)
		}
		prevHash = hash
	}
}

func generateBlock(ctx context.Context, t testing.TB, db *sql.DB, timestamp time.Time, poolTxs []*bc.Tx) error {
	store := txdb.NewStore(db)
	b1, err := store.GetBlock(ctx, 1)
	if err != nil {
		return err
	}
	pool := mempool.New()
	for _, tx := range poolTxs {
		err = pool.Insert(ctx, tx)
		if err != nil {
			return err
		}
	}

	c, err := protocol.NewChain(ctx, b1.Hash(), store, pool, nil)
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	indexer := query.NewIndexer(db, c, pinStore)
	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)

	// Setup the transaction query indexer to index every transaction.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go accounts.ProcessBlocks(ctx)
	indexer.RegisterAnnotator(assets.AnnotateTxs)
	indexer.RegisterAnnotator(accounts.AnnotateTxs)

	block, snapshot, err := c.Recover(ctx)
	if err != nil {
		return err
	}

	b, s, err := c.GenerateBlock(ctx, block, snapshot, timestamp)
	if err != nil {
		return err
	}
	if len(b.Transactions) == 0 {
		return nil
	}
	return c.CommitBlock(ctx, b, s)
}
