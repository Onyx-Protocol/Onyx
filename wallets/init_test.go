package wallets

import (
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"database/sql"
	"log"
	"os"

	"golang.org/x/net/context"
)

var (
	db    *sql.DB
	bgctx context.Context // contains db
)

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	db = pgtest.Open(u, "apitest", "schema.sql")
	bgctx = pg.NewContext(context.Background(), db)
	err := Init(db)
	if err != nil {
		log.Fatal(err)
	}
}
