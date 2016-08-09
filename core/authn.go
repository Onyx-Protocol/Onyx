package core

import (
	chainhttp "chain/net/http"
	"chain/net/http/authn"
	"chain/net/rpc"
)

func rpcAuthn(f chainhttp.HandlerFunc) chainhttp.HandlerFunc {
	return authn.BasicHandler{
		Auth:  rpc.Authenticate,
		Next:  f,
		Realm: "x.chain.com",
	}.ServeHTTPContext
}
