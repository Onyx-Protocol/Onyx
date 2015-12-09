// api-add-user creates user accounts for the api application. The standard
// method of adding user accounts via an invite flow can be inconvenient for
// development purposes, so this tool provides an easy command-line alternative.
//
// api-add-user should be called with two command-line arguments, an email
// address and a password. The database connection can be configured using the
// DB_URL environment variable; the default is to connect to the "api" database
// on localhost.
package main

import (
	"database/sql"
	"log"
	"os"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/database/pg"
	"chain/env"
)

var (
	dbURL = env.String("DB_URL", "postgres:///api?sslmode=disable")
	db    *sql.DB
)

func main() {
	log.SetFlags(0)
	env.Parse()

	if len(os.Args) != 3 {
		log.Fatal("usage: api-add-user email password")
	}

	sql.Register("schemadb", pg.SchemaDriver("api-add-user"))
	db, err := sql.Open("schemadb", *dbURL)
	if err != nil {
		log.Fatalln("error:", err)
	}

	appdb.Init(db)
	ctx := pg.NewContext(context.Background(), db)

	u, err := appdb.CreateUser(ctx, os.Args[1], os.Args[2])
	if err != nil {
		log.Fatalln("error:", err)
	}

	log.Printf("user created: %+v", *u)
}
