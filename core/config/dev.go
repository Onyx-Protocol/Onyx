//+build !prod

package config

import (
	"context"

	"chain/core/mockhsm"
	"chain/crypto/ed25519"
	"chain/database/pg"
	"chain/log"
)

func getOrCreateDevKey(ctx context.Context, db pg.DB, c *Config) (blockPub ed25519.PublicKey, err error) {
	hsm := mockhsm.New(db)
	corePub, created, err := hsm.GetOrCreate(ctx, autoBlockKeyAlias)
	if err != nil {
		return nil, err
	}
	if created {
		log.Messagef(ctx, "Generated new block-signing key %x\n", corePub.Pub)
	} else {
		log.Messagef(ctx, "Using block-signing key %x\n", corePub.Pub)
	}
	c.BlockPub = corePub.Pub

	return corePub.Pub, nil

}

func checkProdBlockHSMURL(_ string) error {
	return nil
}
