package core

import (
	"context"
	"time"

	"chain/core/generator"
	"chain/database/pg"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/vmutil"
)

// ErrProdReset is returned when reset is called on a
// production system.
var ErrProdReset = errors.New("reset called on production system")

func (a *api) reset(ctx context.Context) error {
	lastBlock, err := a.c.LatestBlock(ctx)
	if err != nil {
		return errors.Wrap(err)
	}

	keys, quorum, err := vmutil.ParseBlockMultiSigScript(lastBlock.ConsensusProgram)
	if err != nil {
		return errors.Wrap(err)
	}

	if len(keys) != 0 {
		return ErrProdReset
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
			blocks,
			blocks_txs,
			generator_pending_block,
			issuance_totals,
			leader,
			mockhsm,
			pool_txs,
			query_blocks,
			query_indexes,
			reservations,
			signed_blocks,
			signers,
			snapshots,
			txs
			RESTART IDENTITY;
	`

	_, err = pg.Exec(ctx, q)
	if err != nil {
		return errors.Wrap(err)
	}

	block, err := protocol.NewGenesisBlock(keys, quorum, time.Now())
	if err != nil {
		return errors.Wrap(err)
	}

	err = generator.SaveInitialBlock(ctx, pg.FromContext(ctx), block)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}
