package main

import (
	"context"
	"os"

	"chain/log"
)

// warnCompat prints warnings when it finds an environment variable
// or other option that has been deprecated or removed.
func warnCompat(ctx context.Context) {
	for _, name := range []string{
		"TLSCRT", // read directly by package chain/core
		"TLSKEY", // read directly by package chain/core
	} {
		if os.Getenv(name) != "" {
			log.Printkv(ctx, "warning", "deprecated env var", "name", name)
		}
	}
}
