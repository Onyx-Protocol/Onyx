package api

import (
	"io/ioutil"
	"log"
	"os"

	"chain/api/appdb"
	"chain/database/pg/pgtest"
	chainlog "chain/log"
)

func init() {
	chainlog.SetOutput(ioutil.Discard)

	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	db := pgtest.Open(u, "apitest", "appdb/schema.sql")
	err := appdb.Init(db)
	if err != nil {
		log.Fatal(err)
	}
}
