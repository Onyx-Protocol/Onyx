package pgtest

import (
	stdsql "database/sql"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/lib/pq"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/sql"
	"chain/testutil"
)

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

const DefaultURL = "postgres:///postgres?sslmode=disable"

var (
	DBURL      = os.Getenv("DB_URL_TEST")
	SchemaPath = os.Getenv("CHAIN") + "/api/appdb/schema.sql"
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
func NewContext(t testing.TB) context.Context {
	ctx := context.Background()
	s, err := Open(ctx, DBURL, SchemaPath)
	if err != nil {
		t.Fatal(err)
	}
	runtime.SetFinalizer(s.DB, (*sql.DB).Close)
	ctx = pg.NewContext(ctx, s.DB)
	return ctx
}

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
func Exec(ctx context.Context, t testing.TB, q string) {
	_, err := pg.Exec(ctx, q)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}
