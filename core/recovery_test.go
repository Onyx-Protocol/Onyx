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
	"chain/core/asset/assettest"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/core/txdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/protocol"
	"chain/protocol/bc"
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
	store, pool := txdb.New(db)
	setupCtx := pg.NewContext(ctx, db)
	c, err := assettest.InitializeSigningGenerator(setupCtx, store, pool)
	if err != nil {
		t.Fatal(err)
	}
	// Setup the transaction query indexer to index every transaction.
	indexer := query.NewIndexer(db, c)
	indexer.RegisterAnnotator(account.AnnotateTxs)
	indexer.RegisterAnnotator(asset.AnnotateTxs)

	// Create two assets (USD & apples) and two accounts (Alice & Bob).
	var (
		usdTags = map[string]interface{}{"currency": "usd"}
		usd     = assettest.CreateAssetFixture(setupCtx, t, nil, 0, nil, "usd", usdTags)
		apple   = assettest.CreateAssetFixture(setupCtx, t, nil, 0, nil, "apple", nil)
		alice   = assettest.CreateAccountFixture(setupCtx, t, nil, 0, "alice", nil)
		bob     = assettest.CreateAccountFixture(setupCtx, t, nil, 0, "bob", nil)
	)
	// Issue some apples to Alice and a dollar to Bob.
	_ = assettest.IssueAssetsFixture(setupCtx, t, c, apple, 10, alice)
	_ = assettest.IssueAssetsFixture(setupCtx, t, c, usd, 1, bob)

	prottest.MakeBlock(setupCtx, t, c)

	// Submit a transfer between Alice and Bob but don't publish it in a block.
	assettest.Transfer(setupCtx, t, c, []txbuilder.Action{
		assettest.NewAccountControlAction(bc.AssetAmount{AssetID: usd, Amount: 1}, alice, nil),
		assettest.NewAccountControlAction(bc.AssetAmount{AssetID: apple, Amount: 1}, bob, nil),
		assettest.NewAccountSpendAction(bc.AssetAmount{AssetID: usd, Amount: 1}, bob, nil, nil, nil),
		assettest.NewAccountSpendAction(bc.AssetAmount{AssetID: apple, Amount: 1}, alice, nil, nil, nil),
	})

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
		ctx = pg.NewContext(ctx, wrappedDB)
		go func() {
			err := generateBlock(ctx, wrappedDB, timestamp)
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
			databaseDumps = append(databaseDumps, pgtest.Dump(t, cloneURL, false, "pool_txs"))
			break
		}

		// We crashed at some point during block generation. Do it again,
		// without crashing.
		err = generateBlock(ctx, wrappedDB, timestamp)
		if err != nil {
			t.Fatal(err)
		}
		databaseDumps = append(databaseDumps, pgtest.Dump(t, cloneURL, false, "pool_txs"))
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

func generateBlock(ctx context.Context, db *sql.DB, timestamp time.Time) error {
	store, pool := txdb.New(db)
	c, err := protocol.NewChain(ctx, store, pool, nil)
	if err != nil {
		return err
	}

	asset.Init(c, nil)
	account.Init(c, nil)
	// Setup the transaction query indexer to index every transaction.
	indexer := query.NewIndexer(db, c)
	indexer.RegisterAnnotator(account.AnnotateTxs)
	indexer.RegisterAnnotator(asset.AnnotateTxs)

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
