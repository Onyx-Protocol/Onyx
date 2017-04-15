//+build loopback_auth

package authz

import "context"

// TODO(tessr): Remove this file in favor of localhost grants.

// authorized returns false if this request is unauthorized.
func authorized(ctx context.Context, grants []*Grant) bool {
	return authzToken(ctx, grants) || authzLocalhost(ctx, grants)
}
