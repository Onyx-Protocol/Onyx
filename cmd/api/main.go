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
	chainjson "chain/encoding/json"
	"chain/fedchain/wire"
	"chain/metrics"
	chainhttp "chain/net/http"
	"chain/net/http/gzip"
	"chain/wallet"
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
	wallet.Init(db)

	authAPI := chainhttp.PatServeMux{PatternServeMux: pat.New()}
	authAPI.AddFunc("POST", "/v3/applications/:applicationID/wallets", createWallet)
	authAPI.AddFunc("POST", "/v3/wallets/:walletID/buckets", createBucket)
	authAPI.AddFunc("POST", "/v3/wallets/:walletID/assets", createAsset)
	authAPI.AddFunc("POST", "/v3/assets/:assetID/issue", issueAsset)
	authAPI.AddFunc("POST", "/v3/assets/transfer", walletBuild)
	authAPI.AddFunc("POST", "/v3/wallets/transact/finalize", walletFinalize)

	var h chainhttp.Handler
	h = authAPI // TODO(kr): authentication
	h = metrics.Handler{Handler: h}
	h = gzip.Handler{Handler: h}

	bg := context.Background()
	bg = pg.NewContext(bg, db)
	http.Handle("/", chainhttp.ContextHandler{Context: bg, Handler: h})
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
	asset, err := wallet.AssetByID(ctx, assetID)
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
	writeJSON(w, 200, map[string]interface{}{
		"template": wallet.Tx{
			Unsigned:   buf.Bytes(),
			BlockChain: "sandbox",
			Inputs:     []*wallet.Input{asset.IssuanceInput()},
		},
		"change_addresses": []changeAddr{},
	})
}

// /v3/assets/transfer
func walletBuild(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}

// /v3/wallets/transact/finalize
func walletFinalize(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	// TODO(kr): validate

	var tpl wallet.Tx
	err := json.NewDecoder(req.Body).Decode(&tpl)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	tx := wire.NewMsgTx()
	err = tx.Deserialize(bytes.NewReader(tpl.Unsigned))
	if err != nil {
		w.WriteHeader(400)
		return
	}

	for i, in := range tx.TxIn {
		tplin := tpl.Inputs[i]
		for _, sig := range tplin.Sigs {
			in.SignatureScript = append(in.SignatureScript, sig.DER...)
		}
		in.SignatureScript = append(in.SignatureScript, tplin.RedeemScript...)
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer dbtx.Rollback()

	err = wallet.Commit(ctx, tx)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	err = dbtx.Commit()
	if err != nil {
		w.WriteHeader(500)
		return
	}

	var buf bytes.Buffer
	tx.Serialize(&buf)

	writeJSON(w, 200, map[string]interface{}{
		"transaction_id":  tx.TxSha().String(),
		"raw_transaction": chainjson.HexBytes(buf.Bytes()),
	})
}
