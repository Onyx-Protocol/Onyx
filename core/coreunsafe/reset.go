// Package coreunsafe contains Core logic that is unsafe for
// production.
//
// It is used in the Developer Edition and in command-line
// utilities but shouldn't be used in production.
package coreunsafe

import (
	"context"
	"expvar"

	"chain/database/pg"
	"chain/errors"
)

func isProduction() bool {
	bt := expvar.Get("buildtag")
	return bt != nil && bt.String() != `"dev"`
}

// ResetBlockchain deletes all blockchain data, resulting in an
// unconfigured core. It does not delete access tokens or mockhsm
// keys.
func ResetBlockchain(ctx context.Context, db pg.DB) error {
	if isProduction() {
		// Shouldn't ever happen; This package shouldn't even be
		// included in a production binary.
		panic("reset called on production")
	}

	const q = `
		TRUNCATE
			account_control_programs,
			account_utxos,
			accounts,
			annotated_accounts,
			annotated_assets,
			annotated_outputs,
			annotated_txs,
			asset_tags,
			assets,
			block_processors,
			blocks,
			config,
			generator_pending_block,
			leader,
			pool_txs,
			query_blocks,
			reservations,
			signed_blocks,
			signers,
			snapshots,
			submitted_txs,
			txfeeds
			RESTART IDENTITY;
	`
	_, err := db.Exec(ctx, q)
	return errors.Wrap(err)
}

// ResetEverything deletes all of a Core's data.
func ResetEverything(ctx context.Context, db pg.DB) error {
	if isProduction() {
		// Shouldn't ever happen; This package shouldn't even be
		// included in a production binary.
		panic("reset called on production")
	}

	err := ResetBlockchain(ctx, db)
	if err != nil {
		return errors.Wrap(err)
	}

	const q = `TRUNCATE mockhsm, access_tokens RESTART IDENTITY;`
	_, err = db.Exec(ctx, q)
	return errors.Wrap(err)
}
