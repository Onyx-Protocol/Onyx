package authz

import (
	"context"

	"chain/net/http/authn"
)

func authzToken(ctx context.Context) bool {
	// TODO(tessr): compare against Policies
	return authn.Token(ctx) != ""
}

func authzLocalhost(ctx context.Context) bool {
	// TODO(tessr): compare against Policies
	return authn.Localhost(ctx)
}
