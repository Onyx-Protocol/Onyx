//+build !prod

package main

import (
	"context"
	"log"
	"os"

	"chain/core"
	"chain/database/pg"
	"chain/env"
)

var reset = env.Bool("RESET", false)

func initSchemaInDev(db pg.DB) {
	ctx := context.Background()
	const q = `
		SELECT count(*) FROM pg_tables
		WHERE schemaname='public' AND tablename='migrations'
	`
	var n int
	err := db.QueryRow(ctx, q).Scan(&n)
	if err != nil {
		log.Fatalln("schema init:", err)
	}
	if n == 0 {
		_, err := db.Exec(ctx, core.Schema())
		if err != nil {
			log.Fatalln("schema init:", err)
		}
	}
}

func resetInDevIfRequested(db pg.DB) {
	if *reset {
		os.Setenv("RESET", "false")
		ctx := context.Background()
		err := core.Reset(ctx, db)
		if err != nil {
			log.Fatalln("core reset:", err)
		}
	}
}
