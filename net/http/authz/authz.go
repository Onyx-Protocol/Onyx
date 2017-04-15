//+build !loopback_auth

package authz

import "context"

// authorized returns false if this request is unauthorized.
func authorized(ctx context.Context, grants []*Grant) bool {
	return authzToken(ctx, grants)
}
