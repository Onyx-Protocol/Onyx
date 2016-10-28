package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"expvar"
	"net/http"
	"net/url"
	"strings"
	"time"

	"chain/core/fetch"
	"chain/core/leader"
	"chain/core/mockhsm"
	"chain/core/rpc"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/log"
	"chain/net/http/httpjson"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
)

var (
	errAlreadyConfigured = errors.New("core is already configured; must reset first")
	errUnconfigured      = errors.New("core is not configured")
	errBadGenerator      = errors.New("generator returned an unsuccessful response")
	errBadBlockPub       = errors.New("supplied block pub key is invalid")
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
	networkRPCVersion   = 1
	autoBlockKeyAlias   = "_CHAIN_CORE_AUTO_BLOCK_KEY"
	autoSignReqKeyAlias = "_CHAIN_CORE_AUTO_SIGN_REQ_KEY"
)

func isProduction() bool {
	bt := expvar.Get("buildtag")
	return bt != nil && bt.String() != `"dev"`
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
		err := h.forwardToLeader(ctx, "/info", nil, &resp)
		return resp, err
	}
}

func (h *Handler) leaderInfo(ctx context.Context) (map[string]interface{}, error) {
	var (
		generatorHeight  *uint64
		generatorFetched *time.Time
		snapshot         = fetch.SnapshotProgress()
		localHeight      = h.Chain.Height()
	)
	if h.Config.IsGenerator {
		now := time.Now()
		generatorHeight = &localHeight
		generatorFetched = &now
	} else {
		fetchHeight, fetchTime := fetch.GeneratorHeight()
		// Because everything is asynchronous, it's possible for the localHeight to
		// be higher than our cached generator height. In that case, display the
		// local height as the generator height.
		if localHeight > fetchHeight {
			fetchHeight = localHeight
		}

		// fetchTime might be the zero time if we're having trouble connecting
		// to the remote generator. Only set the height & time if we have it.
		// The dashboard will handle nulls correctly.
		if !fetchTime.IsZero() {
			generatorHeight, generatorFetched = &fetchHeight, &fetchTime
		}
	}

	buildCommit := json.RawMessage(expvar.Get("buildcommit").String())
	buildDate := json.RawMessage(expvar.Get("builddate").String())

	m := map[string]interface{}{
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
		"health":                            h.health(),
	}

	// Add in snapshot information if we're downloading a snapshot.
	if snapshot != nil {
		m["snapshot"] = map[string]interface{}{
			"attempt":     snapshot.Attempt,
			"height":      snapshot.Height,
			"size":        snapshot.Size,
			"downloaded":  snapshot.BytesRead(),
			"in_progress": snapshot.InProgress(),
		}
	}
	return m, nil
}

// Configure configures the core by writing to the database.
// If running in a cored process,
// the caller must ensure that the new configuration is properly reloaded,
// for example by restarting the process.
//
// If c.IsSigner is true, Configure generates a new mockhsm keypair
// for signing blocks, and assigns it to c.BlockPub.
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
		var blockPub ed25519.PublicKey
		if c.BlockPub == "" {
			hsm := mockhsm.New(db)
			corePub, created, err := hsm.GetOrCreate(ctx, autoBlockKeyAlias)
			if err != nil {
				return err
			}
			blockPub = corePub.Pub
			blockPubStr := hex.EncodeToString(blockPub)
			if created {
				log.Messagef(ctx, "Generated new block-signing key %s\n", blockPubStr)
			} else {
				log.Messagef(ctx, "Using block-signing key %s\n", blockPubStr)
			}
			c.BlockPub = blockPubStr
		} else {
			blockPub, err = hex.DecodeString(c.BlockPub)
			if err != nil {
				return err
			}
		}
		signingKeys = append(signingKeys, blockPub)

		if c.IsGenerator {
			if c.SignReqPub == "" {
				hsm := mockhsm.New(db)
				signReqPub, created, err := hsm.GetOrCreate(ctx, autoSignReqKeyAlias)
				if err != nil {
					return err
				}
				signReqPubStr := hex.EncodeToString(signReqPub.Pub)
				if created {
					log.Messagef(ctx, "Generated new sign-request key %s\n", signReqPubStr)
				} else {
					log.Messagef(ctx, "Using sign-request key %s\n", signReqPubStr)
				}
				c.SignReqPub = signReqPubStr
			}
		}
	}

	if c.IsGenerator {
		for _, signer := range c.Signers {
			_, err = url.Parse(signer.URL)
			if err != nil {
				return errors.Wrap(errBadSignerURL, err.Error())
			}
			if len(signer.Pubkey) != ed25519.PublicKeySize {
				return errors.Wrap(errBadSignerPubkey, err.Error())
			}
			signingKeys = append(signingKeys, ed25519.PublicKey(signer.Pubkey))
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

	b := make([]byte, 10)
	_, err = rand.Read(b)
	if err != nil {
		return errors.Wrap(err)
	}
	c.ID = hex.EncodeToString(b)

	const q = `
		INSERT INTO config (id, is_signer, block_pub, sign_request_pub, is_generator,
			blockchain_id, generator_url, generator_access_token,
			remote_block_signers, max_issuance_window_ms, configured_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
	`
	_, err = db.Exec(
		ctx,
		q,
		c.ID,
		c.IsSigner,
		c.BlockPub,
		c.SignReqPub,
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
			SELECT id, is_signer, is_generator,
			blockchain_id, generator_url, generator_access_token, block_pub, sign_request_pub,
			remote_block_signers, max_issuance_window_ms, configured_at
			FROM config
		`

	c := new(Config)
	var (
		blockSignerData []byte
		miw             int64
	)
	err := db.QueryRow(ctx, q).Scan(
		&c.ID,
		&c.IsSigner,
		&c.IsGenerator,
		&c.BlockchainID,
		&c.GeneratorURL,
		&c.GeneratorAccessToken,
		&c.BlockPub,
		&c.SignReqPub,
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
