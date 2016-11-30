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
	"chain/core/pb"
	"chain/core/rpc"
	"chain/core/txdb"
	"chain/crypto/ed25519"
	"chain/database/pg"
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

	Version, BuildCommit, BuildDate string
)

// Config encapsulates Core-level, persistent configuration options.
type Config struct {
	ID                   string
	IsSigner             bool
	IsGenerator          bool
	BlockchainID         bc.Hash
	GeneratorURL         string
	GeneratorAccessToken string
	ConfiguredAt         time.Time
	BlockPub             []byte
	Signers              []BlockSigner
	Quorum               int
	MaxIssuanceWindow    time.Duration
}

type BlockSigner struct {
	AccessToken string
	Pubkey      []byte
	URL         string
}

// Load loads the stored configuration, if any, from the database.
func Load(ctx context.Context, db pg.DB) (*Config, error) {
	const q = `
			SELECT id, is_signer, is_generator,
			blockchain_id, generator_url, generator_access_token, block_pub,
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
		if len(c.BlockPub) == 0 {
			hsm := mockhsm.New(db)
			corePub, created, err := hsm.GetOrCreate(ctx, autoBlockKeyAlias)
			if err != nil {
				return err
			}
			blockPub = corePub.Pub
			if created {
				log.Messagef(ctx, "Generated new block-signing key %x\n", blockPub)
			} else {
				log.Messagef(ctx, "Using block-signing key %x\n", blockPub)
			}
			c.BlockPub = blockPub
		} else {
			blockPub = c.BlockPub
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
		chain, err := protocol.NewChain(ctx, initialBlockHash, store, nil)
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

	const q = `
		INSERT INTO config (id, is_signer, block_pub, is_generator,
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
	conn, err := rpc.NewGRPCConn(url, accessToken, blockchainID, "")
	if err != nil {
		return err
	}
	defer conn.Conn.Close()
	conn.BlockchainID = blockchainID
	resp, err := pb.NewNodeClient(conn.Conn).GetBlockHeight(ctx, nil)
	if err != nil {
		return errors.Wrap(ErrBadGenerator, err.Error())
	}

	if resp.Height < 1 {
		return ErrBadGenerator
	}

	return nil
}
