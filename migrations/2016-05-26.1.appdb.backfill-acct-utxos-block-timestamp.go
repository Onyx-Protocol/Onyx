// +build ignore

package main

import (
	"log"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
)

func main() {
	var dbURL = env.String("DB_URL", "postgres:///core?sslmode=disable")

	env.Parse()

	sql.Register("schemadb", pg.SchemaDriver("backfill-acct-utxos-block-timestamp"))

	db, err := sql.Open("schemadb", *dbURL)
	if err != nil {
		log.Fatal(err)
	}

	ctx := pg.NewContext(context.Background(), db)

	var heights []uint64
	err = pg.ForQueryRows(ctx, "SELECT DISTINCT(confirmed_in) FROM account_utxos WHERE confirmed_in IS NOT NULL ORDER BY confirmed_in", func(h uint64) {
		heights = append(heights, h)
	})
	if err != nil {
		panic(err)
	}
	err = pg.ForQueryRows(ctx, "SELECT data FROM blocks WHERE height IN (SELECT unnest($1::bigint[]))", pg.Uint64s(heights), func(b bc.Block) {
		_, err := pg.Exec(ctx, "UPDATE account_utxos SET block_timestamp = $1 WHERE confirmed_in = $2", b.TimestampMS, b.Height)
		if err != nil {
			panic(err)
		}
	})
	if err != nil {
		panic(err)
	}
}
