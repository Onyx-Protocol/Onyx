package api

import (
	"chain/api/appdb"
	chainhttp "chain/net/http"
	"chain/net/http/authn"
)

func userCredsAuthn(f chainhttp.HandlerFunc) chainhttp.HandlerFunc {
	return authn.BasicHandler{
		Auth:  appdb.AuthenticateUserCreds,
		Next:  f,
		Realm: "x.chain.com",
	}.ServeHTTPContext
}

func tokenAuthn(f chainhttp.HandlerFunc) chainhttp.HandlerFunc {
	return authn.BasicHandler{
		Auth:  appdb.AuthenticateToken,
		Next:  f,
		Realm: "x.chain.com",
	}.ServeHTTPContext
}
