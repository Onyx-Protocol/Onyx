package appdb

import (
	"database/sql"
	"io/ioutil"
	"log"
	"os"

	"chain/database/pg/pgtest"
	chainlog "chain/log"
)

var db *sql.DB

func init() {
	chainlog.SetOutput(ioutil.Discard)

	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	db = pgtest.Open(u, "appdbtest", "schema.sql")
	err := Init(db)
	if err != nil {
		log.Fatal(err)
	}
}
