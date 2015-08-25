package pgtest

import (
	"chain/database/pg"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"testing"

	"github.com/lib/pq"
)

var (
	db     *sql.DB
	schema = "public"
)

// Open creates a sql.DB that is limited to a certain schema.
// This is done by putting a wrapper around the postgres database driver.
// Once the database is opened, Init is called, and the DB is returned.
// dbURI is a standard database connection uri
// schemaName is the name of the database schema to use. It will be created if necessary.
// schemaSQLPath is the filepath to the sql dump of the database
func Open(dbURI, schemaName, schemaSQLPath string) *sql.DB {
	schema = schemaName
	sql.Register("schemadb", pg.SchemaDriver(schemaName))

	var err error
	db, err = sql.Open("schemadb", dbURI)
	if err != nil {
		log.Fatal(err)
	}

	Init(db, schemaSQLPath)

	return db
}

// Init initializes the package to talk to the given database.
// Any SQL statements in file schemaPath
// will be executed before loading each set of fixtures.
// If the db was opened using
func Init(database *sql.DB, schemaSQLPath string) {
	db = database

	const reset = `
		DROP SCHEMA IF EXISTS %s CASCADE;
		CREATE SCHEMA %s;
	`

	quotedSchema := pq.QuoteIdentifier(schema)
	_, err := db.Exec(fmt.Sprintf(reset, quotedSchema, quotedSchema))
	if err != nil {
		panic(err)
	}

	b, err := ioutil.ReadFile(schemaSQLPath)
	if err != nil {
		panic(err)
	}
	q := string(b)
	if schema != "public" {
		q = strings.Replace(q,
			"public, pg_catalog",
			pq.QuoteIdentifier(schema)+", public, pg_catalog",
			-1,
		)
	}
	_, err = db.Exec(q)
	if err != nil {
		panic(err)
	}
}

// ResetWithFile drops all rows from tables
// in the connected database,
// runs the globally-initialized schema SQL,
// then loads the SQL statements in the given file names
// into the database.
func ResetWithFile(t testing.TB, name ...string) {
	var a []string
	for _, s := range name {
		b, err := ioutil.ReadFile(s)
		if err != nil {
			t.Fatal(err)
		}
		a = append(a, string(b))
	}
	ResetWithSQL(t, a...)
}

// ResetWithSQL drops all rows from tables
// in the connected database,
// runs the globally-initialized schema SQL,
// then loads the given SQL statements into the database.
func ResetWithSQL(t testing.TB, sql ...string) {
	clear(t)
	for _, q := range sql {
		exec(t, q)
	}
}

// topSortTables will order tables by foreign key constraints
// so that records can be deleted from tables in order
func topSortTables(tables []string, parents map[string][]string) []string {
	incomingEdges := make(map[string]int)
	for _, pp := range parents {
		for _, p := range pp {
			incomingEdges[p]++
		}
	}

	var insertable []string
	for _, t := range tables {
		if incomingEdges[t] == 0 {
			insertable = append(insertable, t)
		}
	}

	var (
		table string
		fin   []string
	)
	for len(insertable) > 0 {
		table, insertable = insertable[0], insertable[1:]
		for _, p := range parents[table] {
			incomingEdges[p]--
			if incomingEdges[p] == 0 {
				insertable = append(insertable, p)
			}
		}
		fin = append(fin, table)
	}

	return fin
}

func clear(t testing.TB) {
	const getTables = `
		SELECT array_agg(table_name::text) FROM information_schema.tables
		WHERE table_schema=$1 AND table_type='BASE TABLE';
	`
	var tables []string
	err := db.QueryRow(getTables).Scan((*pg.Strings)(&tables))

	const getFkeys = `
		SELECT
     	array_agg(tc.table_name::text), array_agg(ccu.table_name::text)
 		FROM
     	information_schema.table_constraints AS tc
     	JOIN information_schema.key_column_usage
         AS kcu ON tc.constraint_name = kcu.constraint_name
     	JOIN information_schema.constraint_column_usage
         AS ccu ON ccu.constraint_name = tc.constraint_name
		 	WHERE constraint_type = 'FOREIGN KEY';
	`
	var children, parents pg.Strings
	err = db.QueryRow(getFkeys).Scan(&children, &parents)
	if err != nil {
		t.Fatal(err)
	}
	fkeys := make(map[string][]string)
	for i := range children {
		if fkeys[children[i]] == nil {
			fkeys[children[i]] = make([]string, 0, 1)
		}
		fkeys[children[i]] = append(fkeys[children[i]], parents[i])
	}

	tables = topSortTables(tables, fkeys)
	var deletes []string
	for _, t := range tables {
		deletes = append(deletes, "DELETE FROM "+t+";")
	}

	if len(deletes) > 0 {
		exec(t, strings.Join(deletes, "\n"))
	}

	var restarts []string
	rows, err := db.Query("SELECT relname FROM pg_class WHERE relkind = 'S'")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	rq := `ALTER SEQUENCE %s RESTART;`
	for rows.Next() {
		var seq string
		if err = rows.Scan(&seq); err != nil {
			t.Fatal(err)
		}
		restarts = append(restarts, fmt.Sprintf(rq, seq))
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}

	if len(restarts) > 0 {
		exec(t, strings.Join(restarts, "\n"))
	}
}

func exec(t testing.TB, q string) {
	_, err := db.Exec(q)
	if err != nil {
		t.Fatal(err)
	}
}

// Count returns the number of rows in 'table'.
func Count(t *testing.T, table string) int64 {
	var n int64
	err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n)
	if err != nil {
		t.Fatal("Count:", err)
	}
	return n
}
