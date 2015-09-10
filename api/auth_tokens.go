package api

import (
	"chain/api/appdb"
	"chain/net/http/authn"
	"net/http"

	"golang.org/x/net/context"
)

// GET /v3/api-tokens
func listAPITokens(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	uid := authn.GetAuthID(ctx)
	tokens, err := appdb.ListAuthTokens(ctx, uid, "api")
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, tokens)
}

// POST /v3/api-tokens
func createAPIToken(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	uid := authn.GetAuthID(ctx)
	t, err := appdb.CreateAuthToken(ctx, uid, "api", nil)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, t)
}

// DELETE /v3/api-tokens/:tokenID
func deleteAPIToken(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	tid := req.URL.Query().Get(":tokenID")
	err := appdb.DeleteAuthToken(ctx, tid)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, map[string]string{"message": "ok"})
}
