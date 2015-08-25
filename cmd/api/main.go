package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/kr/env"
	"github.com/kr/secureheader"
	"github.com/tessr/pat"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/fedchain/wire"
	"chain/metrics"
	chainhttp "chain/net/http"
	"chain/net/http/gzip"
	"chain/wallets"
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
	wallets.Init(db)

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
	var outs []output
	err := json.NewDecoder(req.Body).Decode(&outs)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	tx := wire.NewMsgTx()
	tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{}))

	assetID := req.URL.Query().Get(":assetID")
	asset, err := wallets.AssetByID(assetID)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	err = addAssetIssuanceOutputs(tx, asset, outs)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	var buf bytes.Buffer
	tx.Serialize(&buf)
	resp := map[string]interface{}{
		"template": wallets.Tx{
			Unsigned:   buf.Bytes(),
			BlockChain: "sandbox",
			Inputs:     []*wallets.Input{asset.IssuanceInput()},
		},
		"change_addresses": []changeAddr{},
	}
	writeJSON(w, resp, 200)
}

// /v3/assets/transfer
func walletBuild(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}

// /v3/wallets/transact/finalize
func walletFinalize(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}
