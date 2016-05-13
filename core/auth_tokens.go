package core

import (
	"chain/core/appdb"
	"chain/net/http/authn"

	"golang.org/x/net/context"
)

// GET /v3/api-tokens
func listAPITokens(ctx context.Context) ([]*appdb.AuthToken, error) {
	uid := authn.GetAuthID(ctx)
	return appdb.ListAuthTokens(ctx, uid, "api")
}

// POST /v3/api-tokens
func createAPIToken(ctx context.Context) (*appdb.AuthToken, error) {
	uid := authn.GetAuthID(ctx)
	return appdb.CreateAuthToken(ctx, uid, "api", nil)
}
