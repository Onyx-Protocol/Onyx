// Package coreunsafe contains Core logic that is unsafe for
// production.
//
// It is used in the Developer Edition and in command-line
// utilities but shouldn't be used in production.
package coreunsafe

import (
	"context"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"chain/core/config"
	"chain/database/pg"
	"chain/errors"
)

var (
	persistBlockchainReset = []string{"mockhsm", "access_tokens"}
	neverReset             = []string{"migrations"}
)

// ResetBlockchain deletes all blockchain data, resulting in an
// unconfigured core. It does not delete access tokens or mockhsm
// keys.
func ResetBlockchain(ctx context.Context, db pg.DB) error {
	if !config.BuildConfig.Reset {
		// Shouldn't ever happen; This package shouldn't even be
		// included in a reset disabled binary.
		panic("reset called on production intended build")
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

	const q = `TRUNCATE %s RESTART IDENTITY;`
	_, err = db.Exec(ctx, fmt.Sprintf(q, strings.Join(tables, ", ")))
	return errors.Wrap(err)
}

// ResetEverything deletes all of a Core's data.
func ResetEverything(ctx context.Context, db pg.DB) error {
	if !config.BuildConfig.Reset {
		// Shouldn't ever happen; This package shouldn't even be
		// included in a reset disabled binary.
		panic("reset called on production intended build")
	}

	err := ResetBlockchain(ctx, db)
	if err != nil {
		return errors.Wrap(err)
	}

	const q = `TRUNCATE %s RESTART IDENTITY;`
	_, err = db.Exec(ctx, fmt.Sprintf(q, strings.Join(persistBlockchainReset, ", ")))
	return errors.Wrap(err)
}
