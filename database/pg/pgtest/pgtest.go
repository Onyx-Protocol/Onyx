package pgtest

import (
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
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/sql"
	"chain/testutil"
)

var (
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	dbpool pool
)

const DefaultURL = "postgres:///postgres?sslmode=disable"

var (
	DBURL      = os.Getenv("DB_URL_TEST")
	SchemaPath = os.Getenv("CHAIN") + "/core/appdb/schema.sql"
)

const (
	gcDur      = 3 * time.Minute
	timeFormat = "20060102150405"
)

type DB struct {
	URL     string // connection string of the form "postgres://..."
	DBName  string // randomized database name
	DB      *sql.DB
	BaseURL string
}

// NewContext calls Open using the background context,
// DBURL, and SchemaPath.
// It puts the resulting sql.DB into a context.
//
// It also registers a finalizer for the DB, so callers
// can discard it without closing it explicitly, and the
// test program is nevertheless unlikely to run out of
// connection slots in the server.
//
// Prefer NewTx whenever the caller can do its
// work in exactly one transaction.
func NewContext(t testing.TB) context.Context {
	ctx := context.Background()
	if os.Getenv("CHAIN") == "" {
		t.Log("warning: $CHAIN not set; probably can't find schema")
	}
	s, err := Open(ctx, DBURL, SchemaPath)
	if err != nil {
		t.Fatal(err)
	}
	runtime.SetFinalizer(s.DB, (*sql.DB).Close)
	ctx = pg.NewContext(ctx, s.DB)
	return ctx
}

// NewTx begins a new transaction on a database
// opened with Open, using DBURL and SchemaPath.
//
// It also registers a finalizer for the Tx, so callers
// can discard it without closing it explicitly, and the
// test program is nevertheless unlikely to run out of
// connection slots in the server.
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
	tx, err := db.DB.Begin(ctx)
	if err != nil {
		db.DB.Close()
		t.Fatal(err)
	}
	// NOTE(kr): we do not set a finalizer on the DB.
	// It is closed explicitly, if necessary, by finalizeTx.
	runtime.SetFinalizer(tx, db.finalizeTx)
	return tx
}

// Open opens a connection to the test database.
// baseURL should be a URL of the form "postgres://.../postgres?...".
// If it is the empty string, DefaultURL will be used.
// The database name component will be replaced with a random name,
// and the resulting URL will be in the returned DB.
func Open(ctx context.Context, baseURL, schemaFile string) (*DB, error) {
	if baseURL == "" {
		baseURL = DefaultURL
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	ctldb, err := stdsql.Open("postgres", baseURL)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	schema, err := ioutil.ReadFile(schemaFile)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(ctx, string(schema))
	if err != nil {
		db.Close()
		return nil, err
	}
	s := &DB{
		URL:     u.String(),
		DBName:  dbname,
		DB:      db,
		BaseURL: baseURL,
	}
	return s, nil
}

func (db *DB) finalizeTx(tx *sql.Tx) {
	ctx := context.Background()
	go func() { // don't block the finalizer goroutine for too long
		err := tx.Rollback(ctx)
		if err != nil {
			// If the tx has been committed (or if anything
			// else goes wrong), we can't reuse db.
			db.DB.Close()
			return
		}
		dbpool.put(db)
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
// as returned from Open. It is the client's job to
// make sure a database is in this state
// (for example, by rolling back a transaction)
// before returning it to the pool.
type pool struct {
	mu  sync.Mutex // protects dbs
	dbs []*DB
}

func (p *pool) get(ctx context.Context, url, path string) (*DB, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.dbs) > 0 {
		db := p.dbs[0]
		p.dbs = p.dbs[1:]
		return db, nil
	}

	return Open(ctx, url, path)
}

func (p *pool) put(db *DB) {
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
