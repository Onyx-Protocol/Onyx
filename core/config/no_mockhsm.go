//+build no_mockhsm

package config

import (
	"context"

	"chain/database/pg"
)

func getOrCreateDevKey(_ context.Context, _ pg.DB, _ *Config) (blockpub []byte, err error) {
	return nil, ErrNoBlockPub
}

func checkProdBlockHSMURL(url string) error {
	if url == "" {
		return ErrNoBlockHSMURL
	}

	return nil
}
