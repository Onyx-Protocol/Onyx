// Package coreunsafe contains Core logic that is unsafe for
// production.
//
// It is used in the Developer Edition and in command-line
// utilities but shouldn't be used in production.
package coreunsafe

import (
	"context"
	"expvar"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/database/raft"
	"chain/errors"
)

var (
	persistBlockchainReset = []string{"mockhsm", "access_tokens"}
	neverReset             = []string{"migrations"}
)

func isProduction() bool {
	p := expvar.Get("prod")
	return p != nil && p.String() == `"yes"`
}

// ResetBlockchain deletes all blockchain data, resulting in an
// unconfigured core. It does not delete access tokens or mockhsm
// keys.
func ResetBlockchain(ctx context.Context, db pg.DB, rDB *raft.Service) error {
	if isProduction() {
		// Shouldn't ever happen; This package shouldn't even be
		// included in a production binary.
		panic("reset called on production")
	}

	var skip []string
	skip = append(skip, persistBlockchainReset...)
	skip = append(skip, neverReset...)

	const tableQ = `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema='public' AND NOT (table_name=ANY($1::text[]))
	`
	var tables []string
	err := pg.ForQueryRows(ctx, db, tableQ, pq.StringArray(skip), func(table string) {
		tables = append(tables, table)
	})
	if err != nil {
		return errors.Wrap(err)
	}

	// Config "table" now lives in raft, and it needs to be deleted too
	err = rDB.Delete(ctx, "/core/config")
	if err != nil {
		return errors.Wrap(err, "could not delete config from RaftDB")
	}

	const q = `TRUNCATE %s RESTART IDENTITY;`
	_, err = db.Exec(ctx, fmt.Sprintf(q, strings.Join(tables, ", ")))
	return errors.Wrap(err)
}

// ResetEverything deletes all of a Core's data.
func ResetEverything(ctx context.Context, db pg.DB, rDB *raft.Service) error {
	if isProduction() {
		// Shouldn't ever happen; This package shouldn't even be
		// included in a production binary.
		panic("reset called on production")
	}

	err := ResetBlockchain(ctx, db, rDB)
	if err != nil {
		return errors.Wrap(err)
	}

	const q = `TRUNCATE %s RESTART IDENTITY;`
	_, err = db.Exec(ctx, fmt.Sprintf(q, strings.Join(persistBlockchainReset, ", ")))
	return errors.Wrap(err)
}
