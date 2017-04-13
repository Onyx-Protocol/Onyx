//+build !loopback_auth

package authz

import (
	"context"
	"log"
)

// Authorized returns false if this request is unauthorized.
func Authorized(ctx context.Context) bool {
	log.Println("non loopback auth")
	return authzToken(ctx)
}
