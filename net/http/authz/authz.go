//+build !loopback_auth

package authz

import "context"

// Authorized returns false if this request is unauthorized.
func Authorized(ctx context.Context) bool {
	return authzToken(ctx)
}
