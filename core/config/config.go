// Package config manages persistent configuration data for
// Chain Core.
package config

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"time"

	"chain/core/rpc"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/sql"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
)

const (
	autoBlockKeyAlias = "_CHAIN_CORE_AUTO_BLOCK_KEY"
)

var (
	ErrBadGenerator      = errors.New("generator returned an unsuccessful response")
	ErrBadSignerURL      = errors.New("block signer URL is invalid")
	ErrBadSignerPubkey   = errors.New("block signer pubkey is invalid")
	ErrBadQuorum         = errors.New("quorum must be greater than 0 if there are signers")
	ErrNoProdBlockPub    = errors.New("blockpub cannot be empty in production")
	ErrNoProdBlockHSMURL = errors.New("block hsm URL cannot be empty in production")

	Version, BuildCommit, BuildDate string
	Production                      bool
)

// Config encapsulates Core-level, persistent configuration options.
type Config struct {
	ID                   string  `json:"id"`
	IsSigner             bool    `json:"is_signer"`
	IsGenerator          bool    `json:"is_generator"`
	BlockchainID         bc.Hash `json:"blockchain_id"`
	GeneratorURL         string  `json:"generator_url"`
	GeneratorAccessToken string  `json:"generator_access_token"`
	BlockHSMURL          string  `json:"block_hsm_url"`
	BlockHSMAccessToken  string  `json:"block_hsm_access_token"`
	ConfiguredAt         time.Time
	BlockPub             string        `json:"block_pub"`
	Signers              []BlockSigner `json:"block_signer_urls"`
	Quorum               int
	MaxIssuanceWindow    chainjson.Duration
}

type BlockSigner struct {
	AccessToken string             `json:"access_token"`
	Pubkey      chainjson.HexBytes `json:"pubkey"`
	URL         string             `json:"url"`
}

// Load loads the stored configuration, if any, from the database.
func Load(ctx context.Context, db pg.DB) (*Config, error) {
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
		&c.BlockHSMURL,
		&c.BlockHSMAccessToken,
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

	c.MaxIssuanceWindow = chainjson.Duration{time.Duration(miw) * time.Millisecond}
	return c, nil
}

// Configure configures the core by writing to the database.
// If running in a cored process,
// the caller must ensure that the new configuration is properly reloaded,
// for example by restarting the process.
//
// When running in non-production mode, if c.IsSigner is true and c.BlockPub is empty,
// Configure generates a new mockhsm keypair
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
		err = checkProdBlockHSMURL(c.BlockHSMURL)
		if err != nil {
			return err
		}
		if c.BlockPub == "" {
			blockPub, err = getOrCreateDevKey(ctx, db, c)
			if err != nil {
				return err
			}
		} else {
			blockPub, err = hex.DecodeString(c.BlockPub)
			if err != nil {
				return err
			}
		}
		signingKeys = append(signingKeys, blockPub)
	}

	if c.IsGenerator {
		for _, signer := range c.Signers {
			_, err = url.Parse(signer.URL)
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

		block, err := protocol.NewInitialBlock(signingKeys, c.Quorum, time.Now())
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

		c.BlockchainID = initialBlockHash
		chain.MaxIssuanceWindow = c.MaxIssuanceWindow.Duration
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
		INSERT INTO config (id, is_signer, block_pub, is_generator,
			blockchain_id, generator_url, generator_access_token,
			block_hsm_url, block_hsm_access_token,
			remote_block_signers, max_issuance_window_ms, configured_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
	`
	_, err = db.Exec(
		ctx,
		q,
		c.ID,
		c.IsSigner,
		c.BlockPub,
		c.IsGenerator,
		c.BlockchainID,
		c.GeneratorURL,
		c.GeneratorAccessToken,
		c.BlockHSMURL,
		c.BlockHSMAccessToken,
		blockSignerData,
		bc.DurationMillis(c.MaxIssuanceWindow.Duration),
	)
	return err
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
