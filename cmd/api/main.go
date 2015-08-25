package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/kr/env"
	"github.com/kr/secureheader"
	"github.com/tessr/pat"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/fedchain/wire"
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
	var outs []struct {
		Address  string
		BucketID string
		Amount   int64
	}
	err := json.NewDecoder(req.Body).Decode(&outs)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	tx := wire.NewMsgTx()
	tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{}))

	aid, err := wire.NewHash20FromStr(req.URL.Query().Get(":assetID"))
	if err != nil {
		w.WriteHeader(400)
		return
	}

	for _, out := range outs {
		if out.BucketID != "" {
			// TODO(erykwalder): actually generate a receiver
			// This address doesn't mean anything, it was grabbed from the internet.
			// We don't have its private key.
			out.Address = "1ByEd6DMfTERyT4JsVSLDoUcLpJTD93ifq"
		}

		addr, err := btcutil.DecodeAddress(out.Address, &chaincfg.MainNetParams)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			w.WriteHeader(400)
			return
		}

		tx.AddTxOut(wire.NewTxOut(aid, out.Amount, pkScript))
	}
}

// /v3/assets/transfer
func walletBuild(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}

// /v3/wallets/transact/finalize
func walletFinalize(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}
