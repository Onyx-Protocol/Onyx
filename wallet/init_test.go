package wallet

import (
	"chain/database/pg/pgtest"
	"database/sql"
	"log"
	"os"
)

var db *sql.DB

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	db = pgtest.Open(u, "apitest", "schema.sql")
	err := Init(db)
	if err != nil {
		log.Fatal(err)
	}
}
