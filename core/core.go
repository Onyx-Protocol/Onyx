package core

import (
	"context"
	"expvar"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"chain/core/fetch"
	"chain/core/generator"
	"chain/core/leader"
	"chain/core/mockhsm"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/net/http/httpjson"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/state"
	"chain/protocol/vmutil"
)

var (
	errAlreadyConfigured = errors.New("core is already configured; must reset first")
	errUnconfigured      = errors.New("core is not configured")
	errBadGenerator      = errors.New("generator returned an unsuccessful response")
	errBadBlockXPub      = errors.New("supplied block xpub is invalid")
)

// reserved mockhsm key alias
const autoBlockKeyAlias = "_CHAIN_CORE_AUTO_BLOCK_KEY"

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
	if a.config == nil {
		// never configured
		return map[string]interface{}{
			"is_configured": false,
		}, nil
	}
	if leader.IsLeading() {
		return a.leaderInfo(ctx)
	} else {
		return a.fetchInfoFromLeader(ctx)
	}
}

func (a *api) leaderInfo(ctx context.Context) (map[string]interface{}, error) {
	localHeight := a.c.Height()
	var (
		generatorHeight  interface{}
		generatorFetched time.Time
	)
	if a.config.IsGenerator {
		generatorHeight = localHeight
		generatorFetched = time.Now()
	} else {
		generatorHeight, generatorFetched = fetch.GeneratorHeight()
	}

	return map[string]interface{}{
		"is_configured":                     true,
		"configured_at":                     a.config.ConfiguredAt,
		"is_signer":                         a.config.IsSigner,
		"is_generator":                      a.config.IsGenerator,
		"generator_url":                     a.config.GeneratorURL,
		"initial_block_hash":                a.config.InitialBlockHash,
		"block_height":                      localHeight,
		"generator_block_height":            generatorHeight,
		"generator_block_height_fetched_at": generatorFetched,
		"is_production":                     expvar.Get("buildtag").String() != "dev",
		"build_commit":                      expvar.Get("buildcommit").String(),
		"build_date":                        expvar.Get("builddate").String(),
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

func configure(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var x Config
	err := httpjson.Read(ctx, r.Body, &x)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	if !x.IsGenerator {
		err = tryGenerator(ctx, x.GeneratorURL)
		if err != nil {
			writeHTTPError(ctx, w, err)
			return
		}
	}

	var signingKeys []ed25519.PublicKey
	if x.IsSigner {
		var blockXPub *hd25519.XPub
		if x.BlockXPub == "" {
			hsm := mockhsm.New(pg.FromContext(ctx))
			coreXPub, created, err := hsm.GetOrCreateKey(ctx, autoBlockKeyAlias)
			if err != nil {
				writeHTTPError(ctx, w, err)
				return
			}
			blockXPub = coreXPub.XPub
			if created {
				log.Printf("Generated new block-signing key %s\n", blockXPub.String())
			} else {
				log.Printf("Using block-signing key %s\n", blockXPub.String())
			}
			x.BlockXPub = blockXPub.String()
		} else {
			blockXPub, err = hd25519.XPubFromString(x.BlockXPub)
			if err != nil {
				writeHTTPError(ctx, w, errors.Wrap(errBadBlockXPub, err.Error()))
				return
			}
		}
		signingKeys = append(signingKeys, blockXPub.Key)
	}

	if x.IsGenerator {
		block, err := protocol.NewGenesisBlock(signingKeys, 1, time.Now())
		if err != nil {
			writeHTTPError(ctx, w, err)
			return
		}
		store, pool := txdb.New(pg.FromContext(ctx).(*sql.DB))
		chain, err := protocol.NewChain(ctx, store, pool, nil)
		if err != nil {
			writeHTTPError(ctx, w, err)
			return
		}

		err = chain.CommitBlock(ctx, block, state.Empty())
		if err != nil {
			writeHTTPError(ctx, w, err)
			return
		}

		x.InitialBlockHash = block.Hash()
	}

	const q = `
		INSERT INTO config (is_signer, block_xpub, is_generator, genesis_hash, generator_url, configured_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
	_, err = pg.Exec(ctx, q, x.IsSigner, x.BlockXPub, x.IsGenerator, x.InitialBlockHash, x.GeneratorURL)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	w.Header().Add("Connection", "close")
	w.WriteHeader(http.StatusOK)
	w.Write(httpjson.DefaultResponse)

	if hijacker, ok := w.(http.Hijacker); ok {
		conn, buf, err := hijacker.Hijack()
		if err != nil {
			log.Printf("could not hijack connection: %s", err)
		} else {
			err = buf.Flush()
			if err != nil {
				log.Printf("could not flush connection buffer: %s", err)
			}
			err = conn.Close()
			if err != nil {
				log.Printf("could not close connection: %s", err)
			}
		}
	}

	execSelf()
}

func tryGenerator(ctx context.Context, url string) error {
	client := &rpc.Client{
		BaseURL: url,
	}
	resp := map[string]interface{}{}
	err := client.Call(ctx, "/rpc/block-height", nil, &resp)
	if err != nil {
		return errors.Wrap(errBadGenerator, err.Error())
	}

	heightField, ok := resp["block_height"]
	if !ok {
		return errBadGenerator
	}

	height, ok := heightField.(uint64)
	if !ok || height < 1 {
		return errBadGenerator
	}

	return nil
}

func execSelf() {
	binpath, err := exec.LookPath(os.Args[0])
	if err != nil {
		panic(err)
	}

	err = syscall.Exec(binpath, os.Args, os.Environ())
	if err != nil {
		panic(err)
	}
}
