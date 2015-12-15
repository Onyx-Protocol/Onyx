package appdb

import (
	"log"
	"os"

	"golang.org/x/net/context"

	"chain/database/pg/pgtest"
	"chain/database/sql"
)

var db *sql.DB

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	db = pgtest.Open(ctx, u, "appdbtest", "schema.sql")
	err := Init(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
}
