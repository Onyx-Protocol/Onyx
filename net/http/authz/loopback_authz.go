//+build loopback_auth

package authz

import (
	"context"

	"chain/net/http/authn"
)

// authorized returns false if this request is unauthorized.
func authorized(ctx context.Context, grants []*Grant) bool {
	return authn.Localhost(ctx) || authzGrants(ctx, grants)
}
