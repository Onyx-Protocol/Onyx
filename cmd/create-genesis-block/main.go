package main

import (
	"database/sql"
	"log"
	"time"

	"chain/api/appdb"
	"chain/database/pg"
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
	appdb.Init(db)

	b := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:   bc.NewBlockVersion,
			Timestamp: uint64(time.Now().Unix()),
		},
	}

	const q = `
		INSERT INTO blocks (block_hash, height, data)
		VALUES ($1, $2, $3)
	`
	_, err = db.Exec(q, b.Hash(), b.Height, b)
	if err != nil {
		log.Fatalln("error:", err)
	}

	log.Printf("block created: %+v", b)
}
