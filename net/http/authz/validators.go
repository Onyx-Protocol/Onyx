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
	// TODO(boymanjor): check if core is configured (add to the context earlier)
	return authn.Localhost(ctx)
}
