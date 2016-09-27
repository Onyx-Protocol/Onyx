package core

import (
	"context"
	"encoding/json"
	"expvar"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"chain/core/fetch"
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
)

var (
	errAlreadyConfigured = errors.New("core is already configured; must reset first")
	errUnconfigured      = errors.New("core is not configured")
	errBadGenerator      = errors.New("generator returned an unsuccessful response")
	errBadBlockXPub      = errors.New("supplied block xpub is invalid")
)

// reserved mockhsm key alias
const (
	networkRPCVersion = 1
	autoBlockKeyAlias = "_CHAIN_CORE_AUTO_BLOCK_KEY"
)

func isProduction() bool {
	return expvar.Get("buildtag").String() != `"dev"`
}

// errProdReset is returned when reset is called on a
// production system.
var errProdReset = errors.New("reset called on production system")

// Reset deletes all data, resulting in an unconfigured core.
// It must be called before any other functions in this package.
func Reset(ctx context.Context, db pg.DB) error {
	if isProduction() {
		return errors.Wrap(errProdReset)
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
			config,
			generator_pending_block,
			leader,
			mockhsm,
			pool_txs,
			query_blocks,
			reservations,
			signed_blocks,
			signers,
			snapshots
			RESTART IDENTITY;
	`

	_, err := db.Exec(ctx, q)
	return errors.Wrap(err)
}

func (a *api) reset(ctx context.Context) error {
	if isProduction() {
		return errors.Wrap(errProdReset)
	}

	w := httpjson.ResponseWriter(ctx)
	closeConnOK(w)
	execSelf("RESET=true")
	panic("unreached")
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
		var resp map[string]interface{}
		err := callLeader(ctx, "/info", nil, &resp)
		return resp, err
	}
}

func (a *api) leaderInfo(ctx context.Context) (map[string]interface{}, error) {
	localHeight := a.c.Height()
	var (
		generatorHeight  uint64
		generatorFetched time.Time
	)
	if a.config.IsGenerator {
		generatorHeight = localHeight
		generatorFetched = time.Now()
	} else {
		generatorHeight, generatorFetched = fetch.GeneratorHeight()
	}

	buildCommit := json.RawMessage(expvar.Get("buildcommit").String())
	buildDate := json.RawMessage(expvar.Get("builddate").String())

	// Because everything is asynchronous, it's possible for the localHeight to
	// be higher than our cached generator height. In that case, display the
	// generatorHeight as our height.
	if localHeight > generatorHeight {
		generatorHeight = localHeight
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
		"is_production":                     isProduction(),
		"network_rpc_version":               networkRPCVersion,
		"build_commit":                      &buildCommit,
		"build_date":                        &buildDate,
	}, nil
}

// Configure configures the core by writing to the database.
// If running in a cored process,
// the caller must ensure that the new configuration is properly reloaded,
// for example by restarting the process.
//
// If c.IsSigner is true, Configure generates a new mockhsm keypair
// for signing blocks, and assigns it to c.BlockXPub.
//
// If c.IsGenerator is true, Configure creates an initial block,
// saves it, and assigns its hash to c.InitialBlockHash.
// Otherwise, c.IsGenerator is false, and Configure makes a test request
// to GeneratorURL to detect simple configuration mistakes.
func Configure(ctx context.Context, db pg.DB, c *Config) error {
	var err error
	if !c.IsGenerator {
		err = tryGenerator(ctx, c.GeneratorURL, c.InitialBlockHash.String())
		if err != nil {
			return err
		}
	}

	var signingKeys []ed25519.PublicKey
	if c.IsSigner {
		var blockXPub *hd25519.XPub
		if c.BlockXPub == "" {
			hsm := mockhsm.New(db)
			coreXPub, created, err := hsm.GetOrCreateKey(ctx, autoBlockKeyAlias)
			if err != nil {
				return err
			}
			blockXPub = coreXPub.XPub
			if created {
				log.Printf("Generated new block-signing key %s\n", blockXPub.String())
			} else {
				log.Printf("Using block-signing key %s\n", blockXPub.String())
			}
			c.BlockXPub = blockXPub.String()
		} else {
			blockXPub, err = hd25519.XPubFromString(c.BlockXPub)
			if err != nil {
				return errors.Wrap(errBadBlockXPub, err.Error())
			}
		}
		signingKeys = append(signingKeys, blockXPub.Key)
	}

	if c.IsGenerator {
		block, err := protocol.NewInitialBlock(signingKeys, 1, time.Now())
		if err != nil {
			return err
		}
		store, pool := txdb.New(db.(*sql.DB))
		chain, err := protocol.NewChain(ctx, store, pool, nil)
		if err != nil {
			return err
		}

		err = chain.CommitBlock(ctx, block, state.Empty())
		if err != nil {
			return err
		}

		c.InitialBlockHash = block.Hash()
	}

	const q = `
		INSERT INTO config (is_signer, block_xpub, is_generator, initial_block_hash, generator_url, configured_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
	_, err = db.Exec(ctx, q, c.IsSigner, c.BlockXPub, c.IsGenerator, c.InitialBlockHash, c.GeneratorURL)
	return err
}

func configure(ctx context.Context, x *Config) error {
	err := Configure(ctx, pg.FromContext(ctx), x)
	if err != nil {
		return err
	}

	w := httpjson.ResponseWriter(ctx)
	closeConnOK(w)
	execSelf()
	panic("unreached")
}

func tryGenerator(ctx context.Context, url, blockchainID string) error {
	client := &rpc.Client{
		BaseURL:      url,
		BlockchainID: blockchainID,
	}
	var x struct {
		BlockHeight uint64 `json:"block_height"`
	}
	err := client.Call(ctx, "/rpc/block-height", nil, &x)
	if err != nil {
		return errors.Wrap(errBadGenerator, err.Error())
	}

	if x.BlockHeight < 1 {
		return errBadGenerator
	}

	return nil
}

func closeConnOK(w http.ResponseWriter) {
	w.Header().Add("Connection", "close")
	w.WriteHeader(http.StatusNoContent)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Printf("no hijacker")
		return
	}
	conn, buf, err := hijacker.Hijack()
	if err != nil {
		log.Printf("could not hijack connection: %s\n", err)
		return
	}
	err = buf.Flush()
	if err != nil {
		log.Printf("could not flush connection buffer: %s\n", err)
	}
	err = conn.Close()
	if err != nil {
		log.Printf("could not close connection: %s\n", err)
	}
}

// execSelf execs Args with environment values replaced
// by the ones in env.
func execSelf(env ...string) {
	binpath, err := exec.LookPath(os.Args[0])
	if err != nil {
		panic(err)
	}

	env = mergeEnvLists(env, os.Environ())
	err = syscall.Exec(binpath, os.Args, env)
	if err != nil {
		panic(err)
	}
}

// mergeEnvLists merges the two environment lists such that
// variables with the same name in "in" replace those in "out".
// This always returns a newly allocated slice.
func mergeEnvLists(in, out []string) []string {
	out = append([]string(nil), out...)
NextVar:
	for _, inkv := range in {
		k := strings.SplitAfterN(inkv, "=", 2)[0]
		for i, outkv := range out {
			if strings.HasPrefix(outkv, k) {
				out[i] = inkv
				continue NextVar
			}
		}
		out = append(out, inkv)
	}
	return out
}
