package pgtest

import (
	"context"
	stdsql "database/sql"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/database/sql"
	"chain/testutil"
)

var (
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	dbpool pool
)

// DefaultURL is used by NewTX and NewDB if DBURL is the empty string.
const DefaultURL = "postgres:///postgres?sslmode=disable"

var (
	// DBURL should be a URL of the form "postgres://...".
	// If it is the empty string, DefaultURL will be used.
	// The functions NewTx and NewDB use it to create and connect
	// to new databases by replacing the database name component
	// with a randomized name.
	DBURL = os.Getenv("DB_URL_TEST")

	// SchemaPath is a file containing a schema to initialize
	// a database in NewTx.
	SchemaPath = os.Getenv("CHAIN") + "/core/appdb/schema.sql"
)

const (
	gcDur      = 3 * time.Minute
	timeFormat = "20060102150405"
)

// NewDB creates a database initialized
// with the schema in schemaPath.
// It returns the resulting *sql.DB with its URL.
//
// It also registers a finalizer for the DB, so callers
// can discard it without closing it explicitly, and the
// test program is nevertheless unlikely to run out of
// connection slots in the server.
//
// Prefer NewTx whenever the caller can do its
// work in exactly one transaction.
func NewDB(t testing.TB, schemaPath string) (url string, db *sql.DB) {
	ctx := context.Background()
	if os.Getenv("CHAIN") == "" {
		t.Log("warning: $CHAIN not set; probably can't find schema")
	}
	url, db, err := open(ctx, DBURL, schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	runtime.SetFinalizer(db, (*sql.DB).Close)
	return url, db
}

// NewTx returns a new transaction on a database
// initialized with the schema in SchemaPath.
//
// It also registers a finalizer for the Tx, so callers
// can discard it without rolling back explicitly, and the
// test program is nevertheless unlikely to run out of
// connection slots in the server.
// The caller should not commit the returned Tx; doing so
// will prevent the underlying database from being reused
// and so cause future calls to NewTx to be slower.
func NewTx(t testing.TB) *sql.Tx {
	runtime.GC() // give the finalizers a better chance to run
	ctx := context.Background()
	if os.Getenv("CHAIN") == "" {
		t.Log("warning: $CHAIN not set; probably can't find schema")
	}
	db, err := dbpool.get(ctx, DBURL, SchemaPath)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	// NOTE(kr): we do not set a finalizer on the DB.
	// It is closed explicitly, if necessary, by finalizeTx.
	runtime.SetFinalizer(tx, finaldb{db}.finalizeTx)
	return tx
}

// open derives a new randomized test database name from baseURL,
// initializes it with schemaFile, and opens it.
func open(ctx context.Context, baseURL, schemaFile string) (newurl string, db *sql.DB, err error) {
	if baseURL == "" {
		baseURL = DefaultURL
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", nil, err
	}

	ctldb, err := stdsql.Open("postgres", baseURL)
	if err != nil {
		return "", nil, err
	}
	defer ctldb.Close()

	err = gcdbs(ctldb)
	if err != nil {
		log.Println(err)
	}

	dbname := pickDBName()
	u.Path = "/" + dbname
	_, err = ctldb.Exec("CREATE DATABASE " + pq.QuoteIdentifier(dbname))
	if err != nil {
		return "", nil, err
	}

	schema, err := ioutil.ReadFile(schemaFile)
	if err != nil {
		return "", nil, err
	}
	db, err = sql.Open("postgres", u.String())
	if err != nil {
		return "", nil, err
	}
	_, err = db.Exec(ctx, string(schema))
	if err != nil {
		db.Close()
		return "", nil, err
	}
	return u.String(), db, nil
}

type finaldb struct{ db *sql.DB }

func (f finaldb) finalizeTx(tx *sql.Tx) {
	ctx := context.Background()
	go func() { // don't block the finalizer goroutine for too long
		err := tx.Rollback(ctx)
		if err != nil {
			// If the tx has been committed (or if anything
			// else goes wrong), we can't reuse db.
			f.db.Close()
			return
		}
		dbpool.put(f.db)
	}()
}

func gcdbs(db *stdsql.DB) error {
	gcTime := time.Now().Add(-gcDur)
	const q = `
		SELECT datname FROM pg_database
		WHERE datname LIKE 'pgtest_%' AND datname < $1
	`
	rows, err := db.Query(q, formatPrefix(gcTime))
	if err != nil {
		return err
	}
	var names []string
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return err
		}
		names = append(names, name)
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	for i, name := range names {
		if i > 5 {
			break // drop up to five databases per test
		}
		go db.Exec("DROP DATABASE " + pq.QuoteIdentifier(name))
	}
	return nil
}

// A pool contains initialized, pristine databases,
// as returned from open. It is the client's job to
// make sure a database is in this state
// (for example, by rolling back a transaction)
// before returning it to the pool.
type pool struct {
	mu  sync.Mutex // protects dbs
	dbs []*sql.DB
}

func (p *pool) get(ctx context.Context, url, path string) (*sql.DB, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.dbs) > 0 {
		db := p.dbs[0]
		p.dbs = p.dbs[1:]
		return db, nil
	}

	_, db, err := open(ctx, url, path)
	return db, err
}

func (p *pool) put(db *sql.DB) {
	p.mu.Lock()
	p.dbs = append(p.dbs, db)
	p.mu.Unlock()
}

func pickDBName() (s string) {
	const chars = "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < 10; i++ {
		s += string(chars[random.Intn(len(chars))])
	}
	return formatPrefix(time.Now()) + s
}

func formatPrefix(t time.Time) string {
	return "pgtest_" + t.UTC().Format(timeFormat) + "Z_"
}

// Exec executes q in the database or transaction in ctx.
// If there is an error, it fails t.
func Exec(ctx context.Context, t testing.TB, q string, args ...interface{}) {
	_, err := pg.Exec(ctx, q, args...)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}
