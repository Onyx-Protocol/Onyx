package authz

import (
	"context"

	"chain/net/http/authn"
)

func authzToken(ctx context.Context, grants []*Grant) bool {
	for _, g := range grants {
		if g.GuardType == "access_token" {
			if accessTokenGuardData(g) == authn.Token(ctx) {
				return true
			}
		}
	}
	return false
}

func authzLocalhost(ctx context.Context, grants []*Grant) bool {
	for _, g := range grants {
		if g.GuardType == "localhost" {
			return true
		}
	}
	return authn.Localhost(ctx)
}

func accessTokenGuardData(grant *Grant) string {
	// retrives id
	return ""
}

// TODO.
// func x509GuardData(grant *Grant) map[string]string {
// 	// retrieves subject map
// 	return nil
// }
