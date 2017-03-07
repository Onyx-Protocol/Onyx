//+build !prod

package config

import (
	"context"
	"encoding/hex"

	"chain/core/mockhsm"
	"chain/database/pg"
	"chain/log"
)

func getOrCreateDevKey(ctx context.Context, db pg.DB, c *Config) (blockPub []byte, err error) {
	hsm := mockhsm.New(db)
	corePub, created, err := hsm.GetOrCreate(ctx, autoBlockKeyAlias)
	if err != nil {
		return nil, err
	}
	blockPub = corePub.Pub
	blockPubStr := hex.EncodeToString(blockPub)
	if created {
		log.Printf(ctx, "Generated new block-signing key %s\n", blockPubStr)
	} else {
		log.Printf(ctx, "Using block-signing key %s\n", blockPubStr)
	}
	c.BlockPub = blockPubStr

	return blockPub, nil

}

func checkProdBlockHSMURL(_ string) error {
	return nil
}
