package api

import (
	"log"
	"os"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/database/pg/pgtest"
)

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	db := pgtest.Open(ctx, u, "apitest", "appdb/schema.sql")
	err := appdb.Init(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
}
