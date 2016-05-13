// api-add-member adds users of the api application to projects. The standard
// method of adding members via an invite flow can be inconvenient for
// development purposes, so this tool provides an easy command-line alternative.
//
// api-add-member should be called with three command-line arguments, an email
// address, a project ID, and a role (either "admin" or "developer"). The
// database connection can be configured using the DB_URL environment variable;
// the default is to connect to the "api" database on localhost.
package main

import (
	"log"
	"os"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
)

var dbURL = env.String("DB_URL", "postgres:///api?sslmode=disable")

func main() {
	log.SetFlags(0)
	env.Parse()

	if len(os.Args) != 4 {
		log.Fatal("usage: api-add-member email projectID role")
	}

	sql.Register("schemadb", pg.SchemaDriver("api-add-member"))
	db, err := sql.Open("schemadb", *dbURL)
	if err != nil {
		log.Fatalln("error:", err)
	}

	ctx := pg.NewContext(context.Background(), db)

	email, projID, role := os.Args[1], os.Args[2], os.Args[3]

	u, err := appdb.GetUserByEmail(ctx, email)
	if err != nil {
		log.Fatalln("error:", err)
	}

	err = appdb.AddMember(ctx, projID, u.ID, role)
	if err != nil {
		log.Fatalln("error:", err)
	}

	log.Printf("%s (%s) added to project %s with role %s", u.Email, u.ID, projID, role)
}
