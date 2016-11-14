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

	"chain/core/mockhsm"
	"chain/core/rpc"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/database/sql"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/log"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/mempool"
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
)

// Config encapsulates Core-level, persistent configuration options.
type Config struct {
	ID                   string  `json:"id"`
	IsSigner             bool    `json:"is_signer"`
	IsGenerator          bool    `json:"is_generator"`
	BlockchainID         bc.Hash `json:"blockchain_id"`
	GeneratorURL         string  `json:"generator_url"`
	GeneratorAccessToken string  `json:"generator_access_token"`
	ConfiguredAt         time.Time
	BlockPub             string        `json:"block_pub"`
	Signers              []BlockSigner `json:"block_signer_urls"`
	Quorum               int
	MaxIssuanceWindow    time.Duration
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
		&c.ID,
		&c.IsSigner,
		&c.IsGenerator,
		&c.BlockchainID,
		&c.GeneratorURL,
		&c.GeneratorAccessToken,
		&c.BlockPub,
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
	}

	if c.IsGenerator {
		for _, signer := range c.Signers {
			_, err = url.Parse(signer.URL)
			if err != nil {
				return errors.Wrap(ErrBadSignerURL, err.Error())
			}
			if len(signer.Pubkey) != ed25519.PublicKeySize {
				return errors.Wrap(ErrBadSignerPubkey, err.Error())
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

		store := txdb.NewStore(db.(*sql.DB))
		chain, err := protocol.NewChain(ctx, initialBlockHash, store, mempool.New(), nil)
		if err != nil {
			return err
		}

		err = chain.CommitBlock(ctx, block, state.Empty())
		if err != nil {
			return err
		}

		c.BlockchainID = initialBlockHash
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

	// TODO(tessr): rename block_xpub column
	const q = `
		INSERT INTO config (id, is_signer, block_xpub, is_generator,
			blockchain_id, generator_url, generator_access_token,
			remote_block_signers, max_issuance_window_ms, configured_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
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
		blockSignerData,
		bc.DurationMillis(c.MaxIssuanceWindow),
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
		return errors.Wrap(ErrBadGenerator, err.Error())
	}

	if x.BlockHeight < 1 {
		return ErrBadGenerator
	}

	return nil
}
