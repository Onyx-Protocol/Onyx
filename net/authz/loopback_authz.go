//+build loopback_auth

package authz

import (
	"context"
	"log"
)

// Authorized returns false if this request is unauthorized.
func Authorized(ctx context.Context) bool {
	log.Println("loopback auth")
	log.Println("wot:", authzLocalhost(ctx))
	return authzToken(ctx) || authzLocalhost(ctx)
}
