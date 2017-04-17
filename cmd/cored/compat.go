package main

import (
	"context"
	"os"

	"chain/log"
)

// warnCompat prints warnings when it finds an environment variable
// or other option that no longer has any meaning.
func warnCompat(ctx context.Context) {
	for _, name := range []string{
		"TLSCRT",
		"TLSKEY",
	} {
		if os.Getenv(name) != "" {
			log.Printkv(ctx, "warning", "unused env var", "name", name)
		}
	}
}
