package core

import (
	"context"
	"expvar"
	"time"

	"chain/core/generator"
	"chain/core/leader"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/vmutil"
)

func getBlockKeys(c *protocol.Chain, ctx context.Context) (keys []ed25519.PublicKey, quorum int, err error) {
	height := c.Height()
	if height == 0 {
		return nil, 0, nil
	}
	lastBlock, err := c.GetBlock(ctx, height)
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

func (a *api) info(ctx context.Context) (map[string]interface{}, error) {
	if leader.IsLeading() {
		return a.leaderInfo(ctx)
	} else {
		return a.fetchInfoFromLeader(ctx)
	}
}

func (a *api) leaderInfo(ctx context.Context) (map[string]interface{}, error) {
	if a.config == nil {
		// never configured
		return map[string]interface{}{
			"is_configured": false,
		}, nil
	}

	localHeight := a.c.Height()
	var generatorHeight interface{}
	if a.config.IsGenerator {
		generatorHeight = localHeight
	} else {
		// TODO(tessr): Store the generator block height in memory on the core leader
		// instead of retrieving it every time.
		generator := &rpc.Client{
			BaseURL: a.config.GeneratorURL,
		}

		var resp map[string]uint64
		err := generator.Call(ctx, "/rpc/block-height", nil, &resp)
		if err != nil {
			log.Error(ctx, err, "could not receive latest block height from generator")
			generatorHeight = "unknown"
		}
		if h, ok := resp["block_height"]; ok {
			generatorHeight = h
		} else {
			log.Write(ctx, "unexpected response from generator")
			generatorHeight = "unknown"
		}
	}

	// TODO(tessr): Add "synced" after SYNC_LIMIT is added.
	return map[string]interface{}{
		"is_configured":          true,
		"configured_at":          a.config.ConfiguredAt,
		"is_signer":              a.config.IsSigner,
		"is_generator":           a.config.IsGenerator,
		"generator_url":          a.config.GeneratorURL,
		"initial_block_hash":     a.config.GenesisHash,
		"block_height":           localHeight,
		"generator_block_height": generatorHeight,
		"is_production":          expvar.Get("buildtag").String() != "dev",
		"build_commit":           expvar.Get("buildcommit").String(),
		"build_date":             expvar.Get("builddate").String(),
	}, nil
}

func (a *api) fetchInfoFromLeader(ctx context.Context) (map[string]interface{}, error) {
	addr, err := leader.Address(ctx)
	if err != nil {
		return nil, err
	}

	l := &rpc.Client{
		BaseURL: "https://" + addr,
		// TODO(tessr): Auth.
	}

	var resp map[string]interface{}
	err = l.Call(ctx, "/info", nil, &resp)
	return resp, err
}
