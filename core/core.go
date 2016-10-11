package core

import (
	"context"
	"encoding/json"
	"expvar"
	"net/http"
	"net/url"
	"strings"
	"time"

	"chain/core/fetch"
	"chain/core/leader"
	"chain/core/mockhsm"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/log"
	"chain/net/http/httpjson"
	"chain/net/rpc"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
)

var (
	errAlreadyConfigured = errors.New("core is already configured; must reset first")
	errUnconfigured      = errors.New("core is not configured")
	errBadGenerator      = errors.New("generator returned an unsuccessful response")
	errBadBlockXPub      = errors.New("supplied block xpub is invalid")
	errNoClientTokens    = errors.New("cannot enable client auth without client access tokens")
	errBadSignerURL      = errors.New("block signer URL is invalid")
	errBadSignerPubkey   = errors.New("block signer pubkey is invalid")
	errBadQuorum         = errors.New("quorum must be greater than 0 if there are signers")
	// errProdReset is returned when reset is called on a
	// production system.
	errProdReset = errors.New("reset called on production system")
)

// reserved mockhsm key alias
const (
	networkRPCVersion = 1
	autoBlockKeyAlias = "_CHAIN_CORE_AUTO_BLOCK_KEY"
)

func isProduction() bool {
	bt := expvar.Get("buildtag")
	return bt != nil && bt.String() != `"dev"`
}

// Schema returns DDL for the entire Chain Core Postgres schema.
func Schema() string {
	return files["schema.sql"]
}

func (h *Handler) reset(ctx context.Context, req struct {
	Everything bool `json:"everything"`
}) error {
	if isProduction() {
		return errors.Wrap(errProdReset)
	}

	dataToReset := "blockchain"
	if req.Everything {
		dataToReset = "everything"
	}

	closeConnOK(httpjson.ResponseWriter(ctx), httpjson.Request(ctx))
	execSelf(dataToReset)
	panic("unreached")
}

func (h *Handler) info(ctx context.Context) (map[string]interface{}, error) {
	if h.Config == nil {
		// never configured
		return map[string]interface{}{
			"is_configured": false,
		}, nil
	}
	if leader.IsLeading() {
		return h.leaderInfo(ctx)
	} else {
		var resp map[string]interface{}
		err := callLeader(ctx, "/info", nil, &resp)
		return resp, err
	}
}

func (h *Handler) leaderInfo(ctx context.Context) (map[string]interface{}, error) {
	localHeight := h.Chain.Height()
	var (
		generatorHeight  uint64
		generatorFetched time.Time
	)
	if h.Config.IsGenerator {
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
		"configured_at":                     h.Config.ConfiguredAt,
		"is_signer":                         h.Config.IsSigner,
		"is_generator":                      h.Config.IsGenerator,
		"generator_url":                     h.Config.GeneratorURL,
		"generator_access_token":            obfuscateTokenSecret(h.Config.GeneratorAccessToken),
		"blockchain_id":                     h.Config.BlockchainID,
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
// saves it, and assigns its hash to c.BlockchainID.
// Otherwise, c.IsGenerator is false, and Configure makes a test request
// to GeneratorURL to detect simple configuration mistakes.
func Configure(ctx context.Context, db pg.DB, c *Config) error {
	var err error
	if !c.IsGenerator {
		err = tryGenerator(
			ctx,
			c.GeneratorURL,
			c.GeneratorAccessToken,
			c.BlockchainID.String(),
		)
		if err != nil {
			return err
		}
	}

	var signingKeys []ed25519.PublicKey
	if c.IsSigner {
		var blockXPub chainkd.XPub
		if c.BlockXPub == "" {
			hsm := mockhsm.New(db)
			coreXPub, created, err := hsm.GetOrCreateChainKDKey(ctx, autoBlockKeyAlias)
			if err != nil {
				return err
			}
			blockXPub = coreXPub.XPub
			if created {
				log.Messagef(ctx, "Generated new block-signing key %s\n", blockXPub.String())
			} else {
				log.Messagef(ctx, "Using block-signing key %s\n", blockXPub.String())
			}
			c.BlockXPub = blockXPub.String()
		} else {
			var blockXPub chainkd.XPub
			err = blockXPub.UnmarshalText([]byte(c.BlockXPub))
			if err != nil {
				return errors.Wrap(errBadBlockXPub, err.Error())
			}
		}
		signingKeys = append(signingKeys, blockXPub.PublicKey())
	}

	if c.IsGenerator {
		for _, signer := range c.Signers {
			_, err = url.Parse(signer.URL)
			if err != nil {
				return errors.Wrap(errBadSignerURL, err.Error())
			}
			signingKey, err := chainkd.NewEd25519PublicKey(signer.Pubkey)
			if err != nil {
				return errors.Wrap(errBadSignerPubkey, err.Error())
			}
			signingKeys = append(signingKeys, signingKey)
		}

		if c.Quorum == 0 && len(signingKeys) > 0 {
			return errors.Wrap(errBadQuorum)
		}

		block, err := protocol.NewInitialBlock(signingKeys, c.Quorum, time.Now())
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

		c.BlockchainID = block.Hash()
		chain.MaxIssuanceWindow = c.MaxIssuanceWindow
	}

	var blockSignerData []byte
	if len(c.Signers) > 0 {
		blockSignerData, err = json.Marshal(c.Signers)
		if err != nil {
			return errors.Wrap(err)
		}
	}

	const q = `
		INSERT INTO config (is_signer, block_xpub, is_generator,
			blockchain_id, generator_url, generator_access_token,
			remote_block_signers, max_issuance_window_ms, configured_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`
	_, err = db.Exec(
		ctx,
		q,
		c.IsSigner,
		c.BlockXPub,
		c.IsGenerator,
		c.BlockchainID,
		c.GeneratorURL,
		c.GeneratorAccessToken,
		blockSignerData,
		bc.DurationMillis(c.MaxIssuanceWindow),
	)
	return err
}

func (h *Handler) configure(ctx context.Context, x *Config) error {
	if h.Config != nil {
		return errAlreadyConfigured
	}

	if x.IsGenerator && x.MaxIssuanceWindow == 0 {
		x.MaxIssuanceWindow = 24 * time.Hour
	}

	err := Configure(ctx, pg.FromContext(ctx), x)
	if err != nil {
		return err
	}

	closeConnOK(httpjson.ResponseWriter(ctx), httpjson.Request(ctx))
	execSelf("")
	panic("unreached")
}

// LoadConfig loads the stored configuration, if any, from the database.
func LoadConfig(ctx context.Context, db pg.DB) (*Config, error) {
	const q = `
			SELECT is_signer, is_generator,
			blockchain_id, generator_url, generator_access_token, block_xpub,
			remote_block_signers, max_issuance_window_ms, configured_at
			FROM config
		`

	c := new(Config)
	var (
		blockSignerData []byte
		miw             int64
	)
	err := db.QueryRow(ctx, q).Scan(
		&c.IsSigner,
		&c.IsGenerator,
		&c.BlockchainID,
		&c.GeneratorURL,
		&c.GeneratorAccessToken,
		&c.BlockXPub,
		&blockSignerData,
		&miw,
		&c.ConfiguredAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "fetching Core config")
	}

	if len(blockSignerData) > 0 {
		err = json.Unmarshal(blockSignerData, &c.Signers)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	c.MaxIssuanceWindow = time.Duration(miw) * time.Millisecond
	return c, nil
}

func tryGenerator(ctx context.Context, url, accessToken, blockchainID string) error {
	client := &rpc.Client{
		BaseURL:      url,
		AccessToken:  accessToken,
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

func closeConnOK(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Connection", "close")
	w.WriteHeader(http.StatusNoContent)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Messagef(req.Context(), "no hijacker")
		return
	}
	conn, buf, err := hijacker.Hijack()
	if err != nil {
		log.Messagef(req.Context(), "could not hijack connection: %s\n", err)
		return
	}
	err = buf.Flush()
	if err != nil {
		log.Messagef(req.Context(), "could not flush connection buffer: %s\n", err)
	}
	err = conn.Close()
	if err != nil {
		log.Messagef(req.Context(), "could not close connection: %s\n", err)
	}
}

func obfuscateTokenSecret(token string) string {
	toks := strings.SplitN(token, ":", 2)
	var res string
	if len(toks) > 0 {
		res += toks[0]
	}
	if len(toks) > 1 {
		res += ":********"
	}
	return res
}
