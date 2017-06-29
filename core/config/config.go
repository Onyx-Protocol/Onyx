// Package config manages persistent configuration data for
// Chain Core.
package config

// Generate code for the Config and BlockSigner types.
//go:generate protoc -I. -I$CHAIN/.. --go_out=. config.proto

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"chain/core/accesstoken"
	"chain/core/rpc"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/sinkdb"
	"chain/errors"
	"chain/log"
	"chain/net/http/authz"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
)

const (
	autoBlockKeyAlias = "_CHAIN_CORE_AUTO_BLOCK_KEY"
	GrantPrefix       = "/core/grant/" // this is also hardcoded in core/authz.go. meh.
)

var (
	ErrBadGenerator    = errors.New("generator returned an unsuccessful response")
	ErrBadSignerURL    = errors.New("block signer URL is invalid")
	ErrBadSignerPubkey = errors.New("block signer pubkey is invalid")
	ErrBadQuorum       = errors.New("quorum must be greater than 0 if there are signers")
	ErrNoBlockPub      = errors.New("blockpub cannot be empty in mockhsm disabled build")
	ErrNoBlockHSMURL   = errors.New("block hsm URL cannot be empty in mockhsm disabled build")
	ErrStaleRaftConfig = errors.New("raft core ID doesn't match Postgres core ID")

	Version, BuildCommit, BuildDate string

	// These feature flags are marked as enabled by build tags.
	// See files in $CHAIN/cmd/cored.
	BuildConfig struct {
		LocalhostAuth bool `json:"is_localhost_auth"`
		MockHSM       bool `json:"is_mockhsm"`
		Reset         bool `json:"is_reset"`
		HTTPOk        bool `json:"is_http_ok"`
		InitCluster   bool `json:"is_init_cluster"`
	}
)

// Load loads the stored configuration, if any, from the database.
// It will first try to load the config from sinkdb; if that fails,
// it will try Postgres next. If it finds a config in Postgres but not in sinkdb
// storage, the config will be added to sinkdb.
func Load(ctx context.Context, db pg.DB, sdb *sinkdb.DB) (*Config, error) {
	c := new(Config)
	ver, err := sdb.Get(ctx, "/core/config", c)
	if err != nil {
		return nil, errors.Wrap(err)
	} else if ver.Exists() {
		var match bool
		match, err = idMatchesPG(ctx, c.Id, db)
		if err != nil {
			return nil, errors.Wrap(err)
		} else if !match {
			raftDir := filepath.Join(HomeDirFromEnvironment(), "raft")
			return nil, errors.Wrap(ErrStaleRaftConfig, "Stale Raft config in "+raftDir)
		}
		return c, nil
	}

	// Check Postgres next.
	c, err = loadFromPG(ctx, db)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	if c == nil {
		return nil, nil
	}

	// If we were able to find a config in Postgres, store it in sinkdb.
	// This also means that we are running this core with raft/sinkdb
	// for the first time which means that we will also migrate access tokens.
	err = sdb.Exec(ctx,
		sinkdb.IfNotExists("/core/config"),
		sinkdb.Set("/core/config", c))
	if err != nil {
		return nil, errors.Wrap(err)
	}
	err = deleteFromPG(ctx, db)
	if err != nil {
		// If we got this far but failed to delete from PG, it's really NBD. Just
		// log the failure and carry on.
		log.Error(ctx, err, "failed to delete config from postgres")
	}
	err = migrateAccessTokens(ctx, db, sdb)
	if err != nil {
		panic(err)
	}
	return c, nil
}

func idMatchesPG(ctx context.Context, id string, db pg.DB) (bool, error) {
	const q = `SELECT id FROM core_id`
	var pgID string
	err := db.QueryRowContext(ctx, q).Scan(&pgID)
	if err != nil && err != sql.ErrNoRows {
		return false, errors.Wrap(err)
	}
	return err == nil && pgID == id, nil
}

// loadFromPG loads the stored configuration from Postgres.
func loadFromPG(ctx context.Context, db pg.DB) (*Config, error) {
	const q = `
			SELECT id, is_signer, is_generator,
			blockchain_id, generator_url, generator_access_token, block_pub,
			block_hsm_url, block_hsm_access_token,
			remote_block_signers, max_issuance_window_ms, configured_at
			FROM config
		`

	c := new(Config)
	var (
		blockSignerData []byte
		blockPubHex     string
		configuredAt    time.Time
	)
	err := db.QueryRowContext(ctx, q).Scan(
		&c.Id,
		&c.IsSigner,
		&c.IsGenerator,
		&c.BlockchainId,
		&c.GeneratorUrl,
		&c.GeneratorAccessToken,
		&blockPubHex,
		&c.BlockHsmUrl,
		&c.BlockHsmAccessToken,
		&blockSignerData,
		&c.MaxIssuanceWindowMs,
		&configuredAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "fetching Core config")
	}

	c.BlockPub, err = hex.DecodeString(blockPubHex)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if len(blockSignerData) > 0 {
		err = json.Unmarshal(blockSignerData, &c.Signers)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	c.ConfiguredAt = bc.Millis(configuredAt)

	return c, nil
}

// deleteFromPG deletes the stored configuration in Postgres.
func deleteFromPG(ctx context.Context, db pg.DB) error {
	// deletes every row
	const q = `DELETE from config`
	_, err := db.ExecContext(ctx, q)
	return errors.Wrap(err, "deleting config stored in postgres")
}

// Loads config status from sinkdb to see if other raft nodes have configured
// so that uncofigured nodes can update.
func CheckConfigExists(ctx context.Context, sdb *sinkdb.DB) (*Config, error) {
	c := new(Config)
	ver, err := sdb.Get(ctx, "/core/config", c)
	if err != nil {
		return nil, errors.Wrap(err)
	} else if ver.Exists() {
		return c, nil
	}
	return nil, nil
}

// Configure configures the core by writing to the database.
// If running in a cored process,
// the caller must ensure that the new configuration is properly reloaded,
// for example by restarting the process.
//
// When running a mockhsm enabled server, if c.IsSigner is true and c.BlockPub is empty,
// Configure generates a new mockhsm keypair
// for signing blocks, and assigns it to c.BlockPub.
//
// If c.IsGenerator is true, Configure creates an initial block,
// saves it, and assigns its hash to c.BlockchainId
// Otherwise, c.IsGenerator is false, and Configure makes a test request
// to GeneratorUrl to detect simple configuration mistakes.
func Configure(ctx context.Context, db pg.DB, sdb *sinkdb.DB, httpClient *http.Client, c *Config) error {
	var err error
	if !c.IsGenerator {
		blockchainID, err := c.BlockchainId.MarshalText()
		err = tryGenerator(
			ctx,
			c.GeneratorUrl,
			c.GeneratorAccessToken,
			string(blockchainID),
			httpClient,
		)
		if err != nil {
			return err
		}
	}

	var signingKeys []ed25519.PublicKey
	if c.IsSigner {
		var blockPub ed25519.PublicKey
		err = checkBlockHSMURL(c.BlockHsmUrl)
		if err != nil {
			return err
		}
		if len(c.BlockPub) == 0 {
			blockPub, err = getOrCreateDevKey(ctx, db, c)
			if err != nil {
				return err
			}
		} else {
			blockPub = ed25519.PublicKey(c.BlockPub)
		}
		signingKeys = append(signingKeys, blockPub)
	}

	// Read the config to ensure that sdb is initialized before
	// we start writing blocks to Postgres.
	// TODO(jackson): make configuration idempotent so that we
	// don't need this.
	ver, err := sdb.Get(ctx, "/core/config", &Config{})
	if err != nil {
		return errors.Wrap(err) // likely uninitialized
	} else if ver.Exists() {
		return errors.Wrap(sinkdb.ErrConflict) // already configured
	}

	if c.IsGenerator {
		for _, signer := range c.Signers {
			_, err = url.Parse(signer.Url)
			if err != nil {
				return errors.Sub(ErrBadSignerURL, err)
			}
			if len(signer.Pubkey) != ed25519.PublicKeySize {
				return errors.Sub(ErrBadSignerPubkey, err)
			}
			signingKeys = append(signingKeys, ed25519.PublicKey(signer.Pubkey))
		}

		if c.Quorum == 0 && len(signingKeys) > 0 {
			return errors.Wrap(ErrBadQuorum)
		}

		block, err := protocol.NewInitialBlock(signingKeys, int(c.Quorum), time.Now())
		if err != nil {
			return err
		}

		initialBlockHash := block.Hash()

		store := txdb.NewStore(db)
		chain, err := protocol.NewChain(ctx, initialBlockHash, store, nil)
		if err != nil {
			return err
		}

		err = chain.CommitAppliedBlock(ctx, block, state.Empty())
		if err != nil {
			return err
		}

		c.BlockchainId = &initialBlockHash
		chain.MaxIssuanceWindow = bc.MillisDuration(c.MaxIssuanceWindowMs)
	}

	b := make([]byte, 10)
	_, err = rand.Read(b)
	if err != nil {
		return errors.Wrap(err)
	}
	c.Id = hex.EncodeToString(b)
	c.ConfiguredAt = bc.Millis(time.Now())

	// Write core ID to postgres to check for matching config between
	// postgres and raft
	const q = `INSERT INTO core_id (id) VALUES ($1)`
	_, err = db.ExecContext(ctx, q, c.Id)
	if err != nil {
		return errors.Wrap(err)
	}

	return sdb.Exec(ctx,
		sinkdb.IfNotExists("/core/config"),
		sinkdb.Set("/core/config", c),
	)
}

func tryGenerator(ctx context.Context, url, accessToken, blockchainID string, httpClient *http.Client) error {
	client := &rpc.Client{
		BaseURL:      url,
		AccessToken:  accessToken,
		BlockchainID: blockchainID,
		Client:       httpClient,
	}
	var x struct {
		BlockHeight uint64 `json:"block_height"`
	}
	err := client.Call(ctx, "/rpc/block-height", nil, &x)
	if err != nil {
		return errors.Sub(ErrBadGenerator, err)
	}

	if x.BlockHeight < 1 {
		return ErrBadGenerator
	}

	return nil
}

// TODO(tessr): make all of this atomic in raft, so we don't get halfway through
// a postgres->raft migration and fail, losing the second half of the migration
func migrateAccessTokens(ctx context.Context, db pg.DB, sdb *sinkdb.DB) error {
	store := authz.NewStore(sdb, GrantPrefix)
	const q = `SELECT id, type, created FROM access_tokens`
	var tokens []*accesstoken.Token
	err := pg.ForQueryRows(ctx, db, q, func(id string, maybeType sql.NullString, created time.Time) {
		t := &accesstoken.Token{
			ID:      id,
			Created: created,
			Type:    maybeType.String,
		}
		tokens = append(tokens, t)
	})

	for _, token := range tokens {
		data := map[string]interface{}{
			"id": token.ID,
		}
		guardData, err := json.Marshal(data)
		if err != nil {
			panic(err) // should never get here
		}

		grant := authz.Grant{
			GuardType: "access_token",
			GuardData: guardData,
			CreatedAt: token.Created.Format(time.RFC3339),
		}
		switch token.Type {
		case "client":
			grant.Policy = "client-readwrite"
		case "network":
			grant.Policy = "crosscore"
		}
		err = sdb.Exec(ctx, store.Save(ctx, &grant))
		if err != nil {
			return errors.Wrap(err)
		}
	}
	return err
}
