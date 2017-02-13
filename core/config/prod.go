//+build prod

package config

import (
	"chain/database/pg"
	"context"
)

func getOrCreateDevKey(_ context.Context, _ pg.DB, _ *Config) (blockpub []byte, err error) {
	return nil, ErrNoProdBlockPub
}
