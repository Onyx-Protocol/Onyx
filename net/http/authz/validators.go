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

func accessTokenGuardData(grant Grant) string {
	// retrives id
	return ""
}

func x509GuardData(grant Grant) map[string]string {
	// retrieves subject map
	return nil
}
