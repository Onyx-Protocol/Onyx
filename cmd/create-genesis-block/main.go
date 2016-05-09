package main

import (
	"encoding/hex"
	"log"

	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/btcec"

	"chain/api/txdb"
	"chain/cos"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
)

var (
	dbURL    = env.String("DB_URL", "postgres:///api?sslmode=disable")
	blockKey = env.String("BLOCK_KEY", "2c1f68880327212b6aa71d7c8e0a9375451143352d5c760dc38559f1159c84ce")
)

func main() {
	log.SetFlags(0)
	env.Parse()

	keyBytes, err := hex.DecodeString(*blockKey)
	if err != nil {
		log.Fatalln("error:", err)
	}

	_, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), keyBytes)

	sql.Register("schemadb", pg.SchemaDriver("create-genesis-block"))
	db, err := sql.Open("schemadb", *dbURL)
	if err != nil {
		log.Fatalln("error:", err)
	}
	ctx := pg.NewContext(context.Background(), db)

	store := txdb.NewStore(db)
	fc, err := cos.NewFC(ctx, store, nil, nil)
	if err != nil {
		log.Fatalln("error:", err)
	}

	b, err := fc.UpsertGenesisBlock(ctx, []*btcec.PublicKey{pubKey}, 1)
	if err != nil {
		log.Fatalln("error:", err)
	}
	log.Printf("block created: %+v", b)
}
