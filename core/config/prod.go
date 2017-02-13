//+build prod

package config

import (
	"context"

	"chain/database/pg"
)

func getOrCreateDevKey(_ context.Context, _ pg.DB, _ *Config) (blockpub []byte, err error) {
	return nil, ErrNoProdBlockPub
}
