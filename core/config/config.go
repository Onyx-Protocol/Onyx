// Package config manages persistent configuration data for
// Chain Core.
package config

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"time"

	"github.com/golang/protobuf/proto"

	"chain/core/mockhsm"
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
	"chain/protocol/mempool"
	"chain/protocol/state"
)

//go:generate protoc -I. -I$CHAIN/.. --go_out=. config.proto

const (
	autoBlockKeyAlias = "_CHAIN_CORE_AUTO_BLOCK_KEY"
)

var (
	ErrBadGenerator    = errors.New("generator returned an unsuccessful response")
	ErrBadSignerURL    = errors.New("block signer URL is invalid")
	ErrBadSignerPubkey = errors.New("block signer pubkey is invalid")
	ErrBadQuorum       = errors.New("quorum must be greater than 0 if there are signers")
)

// Load loads the stored configuration, if any, from the database.
func Load(ctx context.Context, db pg.DB, rDB *raft.Service) (*Config, error) {
	// We do a stale read followed by a linearizable read.
	// We can't do a linearizable read if this is a new node in a preexisting
	// network because this node isn't listening for http requests yet; so we
	// do a stale read instead.
	// However, if this is a freshly configured node in a fresh network, we can't
	// do a stale read because we will miss the newly created node configuration.
	// So in that case, we must do a linearizable read.
	data := rDB.Stale().Get("/core/config")
	var err error
	if data == nil {
		data, err = rDB.Get(ctx, "/core/config")
		if err != nil {
			return nil, errors.Wrap(err)
		}
		if data == nil {
			return nil, nil
		}
	}

	c := new(Config)
	err = proto.Unmarshal(data, c)
	if err != nil {
		return nil, errors.Wrap(err)
	}
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
// saves it, and assigns its hash to c.BlockchainId
// Otherwise, c.IsGenerator is false, and Configure makes a test request
// to GeneratorUrl to detect simple configuration mistakes.
func Configure(ctx context.Context, db pg.DB, rDB *raft.Service, c *Config) error {
	var err error
	if !c.IsGenerator {
		err = tryGenerator(
			ctx,
			c.GeneratorUrl,
			c.GeneratorAccessToken,
			c.BlockchainId.Hash().String(),
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
				log.Messagef(ctx, "Generated new block-signing key %x\n", corePub.Pub)
			} else {
				log.Messagef(ctx, "Using block-signing key %x\n", corePub.Pub)
			}
			c.BlockPub = corePub.Pub
		}
		signingKeys = append(signingKeys, blockPub)
	}

	if c.IsGenerator {
		for _, signer := range c.Signers {
			_, err = url.Parse(signer.Url)
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

		block, err := protocol.NewInitialBlock(signingKeys, int(c.Quorum), time.Now())
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

		c.BlockchainId = initialBlockHash.Proto()
		//ToDO: implement in bc/time.go
		chain.MaxIssuanceWindow = bc.MillisDuration(c.MaxIssuanceWindow)
	}

	b := make([]byte, 10)
	_, err = rand.Read(b)
	if err != nil {
		return errors.Wrap(err)
	}
	c.Id = hex.EncodeToString(b)

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
		return errors.Wrap(ErrBadGenerator, err.Error())
	}

	if x.BlockHeight < 1 {
		return ErrBadGenerator
	}

	return nil
}
