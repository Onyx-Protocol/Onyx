package appdb

import (
	"chain/database/pg/pgtest"
	"database/sql"
	"io/ioutil"
	"log"
	"os"
)

var db *sql.DB

func init() {
	log.SetOutput(ioutil.Discard)

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
