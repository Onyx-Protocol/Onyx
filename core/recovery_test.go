package core

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/lib/pq"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
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
	c := prottest.NewChainWithStorage(t, store, pool)
	indexer := query.NewIndexer(db, c)
	assets := asset.NewRegistry(c, bc.Hash{})
	accounts := account.NewManager(c)
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)

	// Setup the transaction query indexer to index every transaction.
	indexer.RegisterAnnotator(accounts.AnnotateTxs)
	indexer.RegisterAnnotator(assets.AnnotateTxs)
	indexer.IndexTransactions()

	// Create two assets (USD & apples) and two accounts (Alice & Bob).
	var (
		usdTags = map[string]interface{}{"currency": "usd"}
		usd     = coretest.CreateAsset(setupCtx, t, assets, nil, "usd", usdTags)
		apple   = coretest.CreateAsset(setupCtx, t, assets, nil, "apple", nil)
		alice   = coretest.CreateAccount(setupCtx, t, accounts, "alice", nil)
		bob     = coretest.CreateAccount(setupCtx, t, accounts, "bob", nil)
	)
	// Issue some apples to Alice and a dollar to Bob.
	_ = coretest.IssueAssets(setupCtx, t, c, assets, accounts, apple, 10, alice)
	_ = coretest.IssueAssets(setupCtx, t, c, assets, accounts, usd, 1, bob)

	prottest.MakeBlock(setupCtx, t, c)

	// Submit a transfer between Alice and Bob but don't publish it in a block.
	coretest.Transfer(setupCtx, t, c, []txbuilder.Action{
		accounts.NewControlAction(bc.AssetAmount{AssetID: usd, Amount: 1}, alice, nil),
		accounts.NewControlAction(bc.AssetAmount{AssetID: apple, Amount: 1}, bob, nil),
		accounts.NewSpendAction(bc.AssetAmount{AssetID: usd, Amount: 1}, bob, nil, nil, nil, nil),
		accounts.NewSpendAction(bc.AssetAmount{AssetID: apple, Amount: 1}, alice, nil, nil, nil, nil),
	})

	// Save a copy of the pool txs
	var poolTxs []*bc.TxData
	err := pg.ForQueryRows(setupCtx, `SELECT data FROM pool_txs`, func(tx bc.TxData) {
		poolTxs = append(poolTxs, &tx)
	})
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
			databaseDumps = append(databaseDumps, pgtest.Dump(t, cloneURL, false, "pool_txs", "*_id_seq"))
			break
		}

		// At some point, generateBlock deletes the contents of the tx
		// pool. If it crashes at that point, those txs are lost and
		// recovery won't produce the same output as on all the other
		// runs. In a running network this isn't too big a deal because
		// the submitters of the pool txs will resubmit them if they fail
		// to appear in a block. We simulate that in this case by
		// replacing the deleted pool txs before trying to recover.

		hashes := make([]string, 0, len(poolTxs))
		txstrs := make([][]byte, 0, len(poolTxs))
		for _, poolTx := range poolTxs {
			hashes = append(hashes, poolTx.Hash().String())
			txstr, err := poolTx.Value()
			if err != nil {
				t.Fatal(err)
			}
			txstrs = append(txstrs, txstr.([]byte))
		}
		_, err = wrappedDB.Exec(ctx, `INSERT INTO pool_txs (tx_hash, data) VALUES (unnest($1::text[]), unnest($2::bytea[])) ON CONFLICT (tx_hash) DO NOTHING`, pq.StringArray(hashes), pq.ByteaArray(txstrs))
		if err != nil {
			t.Fatal(err)
		}

		// We crashed at some point during block generation. Do it again,
		// without crashing.
		err = generateBlock(ctx, wrappedDB, timestamp)
		if err != nil {
			t.Fatal(err)
		}
		databaseDumps = append(databaseDumps, pgtest.Dump(t, cloneURL, false, "pool_txs", "*_id_seq"))
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
	indexer := query.NewIndexer(db, c)

	initial, err := c.GetBlock(ctx, 1)
	if err != nil {
		return err
	}

	assets := asset.NewRegistry(c, initial.Hash())
	accounts := account.NewManager(c)

	// Setup the transaction query indexer to index every transaction.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	indexer.RegisterAnnotator(assets.AnnotateTxs)
	indexer.RegisterAnnotator(accounts.AnnotateTxs)
	indexer.IndexTransactions()

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
