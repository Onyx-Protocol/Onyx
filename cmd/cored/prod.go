//+build prod

package main

import (
	"context"

	"chain/log"
)

func requireSecretInProd(secret string) {
	if secret == "" {
		ctx := context.Background()
		log.Fatal(ctx, "error", "please set environment variable API_SECRET")
	}
}
