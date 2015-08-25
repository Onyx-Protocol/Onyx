package wallets

import (
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"log"
	"os"
)

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	db = pgtest.Open(u, "apitest", "schema.sql")

	err := pg.LoadFile(db, "reserve.sql", "keys.sql")
	if err != nil {
		log.Fatal(err)
	}
}
