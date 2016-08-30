package core

import (
	"context"
	"time"

	"chain/core/generator"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/vmutil"
)

func getBlockKeys(c *protocol.Chain, ctx context.Context) (keys []ed25519.PublicKey, quorum int, err error) {
	lastBlock, err := c.LatestBlock(ctx)
	if err == protocol.ErrNoBlocks {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}
	return vmutil.ParseBlockMultiSigScript(lastBlock.ConsensusProgram)
}

// errProdReset is returned when reset is called on a
// production system.
var errProdReset = errors.New("reset called on production system")

func (a *api) reset(ctx context.Context) error {
	keys, quorum, err := getBlockKeys(a.c, ctx)
	if err != nil {
		return errors.Wrap(err)
	}

	if len(keys) != 0 {
		return errProdReset
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

	// Reset the height on the blockchain.
	a.c.Reset()

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
