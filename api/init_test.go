package api

import (
	"chain/api/appdb"
	"chain/database/pg/pgtest"
	"io/ioutil"
	"log"
	"os"
)

func init() {
	log.SetOutput(ioutil.Discard)

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
