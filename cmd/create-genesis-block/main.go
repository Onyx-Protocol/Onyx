package main

import (
	"log"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
)

var (
	dbURL = env.String("DB_URL", "postgres:///api?sslmode=disable")
	db    *sql.DB
)

func main() {
	log.SetFlags(0)
	env.Parse()

	sql.Register("schemadb", pg.SchemaDriver("create-genesis-block"))
	db, err := sql.Open("schemadb", *dbURL)
	if err != nil {
		log.Fatalln("error:", err)
	}
	ctx := pg.NewContext(context.Background(), db)
	appdb.Init(ctx, db)

	b, err := asset.UpsertGenesisBlock(ctx)
	if err != nil {
		log.Fatalln("error:", err)
	}
	log.Printf("block created: %+v", b)
}
