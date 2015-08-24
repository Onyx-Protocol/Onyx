package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/kr/env"
	"github.com/kr/secureheader"
	"github.com/tessr/pat"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/metrics"
	chainhttp "chain/net/http"
	"chain/net/http/gzip"
)

var (
	// config vars
	listenAddr = env.String("LISTEN", ":8080")
	dbURL      = env.String("DB_URL", "postgres:///api?sslmode=disable")

	db       *sql.DB
	buildTag = "dev"
)

func main() {
	sql.Register("schemadb", pg.SchemaDriver(buildTag))
	log.SetPrefix("api-" + buildTag + ": ")
	log.SetFlags(log.Lshortfile)
	env.Parse()

	var err error
	db, err = sql.Open("schemadb", *dbURL)
	if err != nil {
		log.Fatal(err)
	}

	pg.LoadFile(db, "reserve.sql")

	authAPI := chainhttp.PatServeMux{pat.New()}
	authAPI.AddFunc("POST", "/v3/applications/:applicationID/wallets", createWallet)
	authAPI.AddFunc("POST", "/v3/wallets/:walletID/buckets", createBucket)
	authAPI.AddFunc("POST", "/v3/wallets/:walletID/assets", createAsset)
	authAPI.AddFunc("POST", "/v3/assets/:assetID/issue", issueAsset)
	authAPI.AddFunc("POST", "/v3/assets/transfer", walletBuild)
	authAPI.AddFunc("POST", "/v3/wallets/transact/finalize", walletFinalize)

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
