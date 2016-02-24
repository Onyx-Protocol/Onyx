package api

import (
	"os"

	"golang.org/x/net/context"

	"chain/database/pg/pgtest"
)

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	pgtest.Open(ctx, u, "apitest", "appdb/schema.sql")
}
