package main

import (
	"log"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
	"chain/fedchain/bc"
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

	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:   bc.NewBlockVersion,
			Timestamp: uint64(time.Now().Unix()),
		},
	}

	const q = `
		INSERT INTO blocks (block_hash, height, data, header)
		VALUES ($1, $2, $3, $4)
	`
	_, err = db.Exec(ctx, q, b.Hash(), b.Height, b, &b.BlockHeader)
	if err != nil {
		log.Fatalln("error:", err)
	}

	log.Printf("block created: %+v", b)
}
