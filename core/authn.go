package core

import (
	"golang.org/x/net/context"

	chainhttp "chain/net/http"
	"chain/net/http/authn"
	"chain/net/rpc"
	"crypto/subtle"
)

func rpcAuthn(f chainhttp.HandlerFunc) chainhttp.HandlerFunc {
	return authn.BasicHandler{
		Auth:  rpc.Authenticate,
		Next:  f,
		Realm: "x.chain.com",
	}.ServeHTTPContext
}

func apiAuthn(secret string, next chainhttp.HandlerFunc) chainhttp.HandlerFunc {
	// If the secret is blank, we should not require an HTTP Basic Auth header,
	// nor should we present a WWW-Authenticate challenge.
	if secret == "" {
		return next
	}

	authFunc := func(ctx context.Context, _, pw string) error {
		if subtle.ConstantTimeCompare([]byte(pw), []byte(secret)) == 0 {
			return authn.ErrNotAuthenticated
		}
		return nil
	}

	return authn.BasicHandler{
		Auth:  authFunc,
		Next:  next,
		Realm: "Chain Core API",
	}.ServeHTTPContext
}
