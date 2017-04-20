// Package config manages persistent configuration data for
// Chain Core.
package config

// Generate code for the Config and BlockSigner types.
//go:generate protoc -I. -I$CHAIN/.. --go_out=. config.proto

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"time"

	"github.com/golang/protobuf/proto"

	"chain/core/rpc"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/raft"
	"chain/database/sql"
	"chain/errors"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
)

const (
	autoBlockKeyAlias = "_CHAIN_CORE_AUTO_BLOCK_KEY"
)

var (
	ErrBadGenerator    = errors.New("generator returned an unsuccessful response")
	ErrBadSignerURL    = errors.New("block signer URL is invalid")
	ErrBadSignerPubkey = errors.New("block signer pubkey is invalid")
	ErrBadQuorum       = errors.New("quorum must be greater than 0 if there are signers")
	ErrNoBlockPub      = errors.New("blockpub cannot be empty in mockhsm disabled build")
	ErrNoBlockHSMURL   = errors.New("block hsm URL cannot be empty in mockhsm disabled build")

	Version, BuildCommit, BuildDate string

	// This is the default, Chain development configuration.
	// These options can be updated with build tags.
	BuildConfig = struct {
		LoopbackAuth bool `json:"is_loopback_auth"`
		MockHSM      bool `json:"is_mockhsm"`
		Reset        bool `json:"is_reset"`
	}{
		LoopbackAuth: false,
		MockHSM:      true,
		Reset:        false,
	}
)

// Load loads the stored configuration, if any, from the database.
// It will first try to load the config from raft storage; if that fails,
// it will try Postgres next. If it finds a config in Postgres but not in raft
// storage, the config will be added to raft storage.
func Load(ctx context.Context, db pg.DB, rDB *raft.Service) (*Config, error) {
	data, err := rDB.Get(ctx, "/core/config")
	if err != nil {
		return nil, errors.Wrap(err)
	}
	if data == nil {
		// Check Postgres next.
		config, err := loadFromPG(ctx, db)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		// If we were able to find a config in Postgres, store it in Raft.
		// This also means that we are running this core with raft for the first time
		// which means that we will also migrate access tokens.
		if config != nil {
			val, err := proto.Marshal(config)
			if err != nil {
				return nil, errors.Wrap(err)
			}
			err = rDB.Insert(ctx, "/core/config", val)
			if err != nil {
				return nil, errors.Wrap(err)
			}
			err = deleteFromPG(ctx, db)
			if err != nil {
				// If we got this far but failed to delete from PG, it's really NBD. Just
				// log the failure and carry on.
				log.Error(ctx, err, "failed to delete config from postgres")
			}
			err = migrateAccessTokens(ctx, db, rDB)
			return config, nil
		}
		return nil, nil
	}

	c := new(Config)
	err = proto.Unmarshal(data, c)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return c, nil
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
	err := db.QueryRow(ctx, q).Scan(
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
	_, err := db.Exec(ctx, q)
	return errors.Wrap(err, "deleting config stored in postgres")
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
func Configure(ctx context.Context, db pg.DB, rDB *raft.Service, c *Config) error {
	var err error
	if !c.IsGenerator {
		blockchainID, err := c.BlockchainId.MarshalText()
		err = tryGenerator(
			ctx,
			c.GeneratorUrl,
			c.GeneratorAccessToken,
			string(blockchainID),
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

	val, err := proto.Marshal(c)
	if err != nil {
		return errors.Wrap(err)
	}
	return rDB.Insert(ctx, "/core/config", val)
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
		return errors.Sub(ErrBadGenerator, err)
	}

	if x.BlockHeight < 1 {
		return ErrBadGenerator
	}

	return nil
}

// this almost certainly should live in another package
func migrateAccessTokens(ctx context.Context, db *pg.DB, rDB *raft.Service) error {
	const q = `SELECT id, type, sort_id, created FROM access_tokens`
	var tokens []*accesstoken.Token
	err := pg.ForQueryRows(ctx, cs.DB, q, func(id string, maybeType sql.NullString, sortID string, created time.Time) {
		t := Token{
			ID:      id,
			Created: created,
			Type:    maybeType.String,
			sortID:  sortID,
		}
		tokens = append(tokens, &t)
	})
	if err != nil {
		return nil, "", errors.Wrap(err)
	}
}
