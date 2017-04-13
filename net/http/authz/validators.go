package authz

import (
	"context"

	"chain/net/http/authn"
)

func authzToken(ctx context.Context) bool {
	// TODO(tessr): compare against Policies
	_, ok := authn.TokenFromContext(ctx)
	return ok
}

func authzLocalhost(ctx context.Context) bool {
	// TODO(tessr): compare against Policies
	return authn.LocalhostFromContext(ctx)
}
