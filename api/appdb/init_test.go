package appdb_test

import (
	"os"

	"golang.org/x/net/context"

	"chain/api/txdb"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/fedchain"
)

var (
	db    *sql.DB
	store fedchain.Store
)

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	db = pgtest.Open(ctx, u, "appdbtest", "schema.sql")
	store = txdb.NewStore()
}
