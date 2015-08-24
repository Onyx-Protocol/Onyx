package main

import (
	"net/http"

	"github.com/kr/env"
	"github.com/kr/secureheader"
	"github.com/tessr/pat"

	"chain/metrics"
	"chain/net/http/gzip"
)

// config vars
var (
	listenAddr = env.String("LISTEN", ":8080")
)

func main() {
	authAPI := patServeMux{pat.New()}
	authAPI.Add("POST", "/v3/applications/:applicationID/wallets", handlerFunc(createWallet))
	authAPI.Add("POST", "/v3/wallets/:walletID/buckets", handlerFunc(createBucket))
	authAPI.Add("POST", "/v3/wallets/:walletID/assets", handlerFunc(createAsset))
	authAPI.Add("POST", "/v3/assets/:assetID/issue", handlerFunc(issueAsset))
	authAPI.Add("POST", "/v3/assets/transfer", handlerFunc(walletBuild))
	authAPI.Add("POST", "/v3/wallets/transact/finalize", handlerFunc(walletFinalize))

	var h handler
	h = authAPI // TODO(kr): authentication
	h = metrics.Handler{h}
	h = gzip.Handler{h}

	http.Handle("/", h)
	http.HandleFunc("/health", func(http.ResponseWriter, *http.Request) {})

	secureheader.DefaultConfig.PermitClearLoopback = true
	http.ListenAndServe(*listenAddr, secureheader.DefaultConfig)
}
