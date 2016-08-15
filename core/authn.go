package core

import (
	"crypto/subtle"

	"golang.org/x/net/context"

	chainhttp "chain/net/http"
	"chain/net/http/authn"
	"chain/net/rpc"
)

func rpcAuthn(f chainhttp.Handler) chainhttp.Handler {
	return authn.BasicHandler{
		Auth:  rpc.Authenticate,
		Next:  f,
		Realm: "x.chain.com",
	}
}

func apiAuthn(secret string, next chainhttp.Handler) chainhttp.Handler {
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
	}
}
