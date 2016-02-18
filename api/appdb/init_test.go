package appdb_test

import (
	"os"

	"golang.org/x/net/context"

	"chain/api/asset"
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
	store = txdb.NewStore()
	fc := fedchain.New(store, nil)
	asset.Init(fc, nil, true)
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	db = pgtest.Open(ctx, u, "appdbtest", "schema.sql")
}
