package main

import (
	"net/http"

	"github.com/kr/env"
	"github.com/kr/secureheader"
	"github.com/tessr/pat"
	"golang.org/x/net/context"

	"chain/metrics"
	chainhttp "chain/net/http"
	"chain/net/http/gzip"
)

// config vars
var (
	listenAddr = env.String("LISTEN", ":8080")
)

func main() {
	authAPI := chainhttp.PatServeMux{pat.New()}
	authAPI.Add("POST", "/v3/applications/:applicationID/wallets", chainhttp.HandlerFunc(createWallet))
	authAPI.Add("POST", "/v3/wallets/:walletID/buckets", chainhttp.HandlerFunc(createBucket))
	authAPI.Add("POST", "/v3/wallets/:walletID/assets", chainhttp.HandlerFunc(createAsset))
	authAPI.Add("POST", "/v3/assets/:assetID/issue", chainhttp.HandlerFunc(issueAsset))
	authAPI.Add("POST", "/v3/assets/transfer", chainhttp.HandlerFunc(walletBuild))
	authAPI.Add("POST", "/v3/wallets/transact/finalize", chainhttp.HandlerFunc(walletFinalize))

	var h chainhttp.Handler
	h = authAPI // TODO(kr): authentication
	h = metrics.Handler{h}
	h = gzip.Handler{h}

	http.Handle("/", chainhttp.BackgroundHandler{h})
	http.HandleFunc("/health", func(http.ResponseWriter, *http.Request) {})

	secureheader.DefaultConfig.PermitClearLoopback = true
	http.ListenAndServe(*listenAddr, secureheader.DefaultConfig)
}

// /v3/applications/:applicationID/wallets
func createWallet(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}

// /v3/wallets/:walletID/buckets
func createBucket(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}

// /v3/wallets/:walletID/assets
func createAsset(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}

// /v3/assets/:assetID/issue
func issueAsset(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}

// /v3/assets/transfer
func walletBuild(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}

// /v3/wallets/transact/finalize
func walletFinalize(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}
