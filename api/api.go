// Package api provides http handlers for all Chain operations.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/tessr/pat"
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/fedchain/wire"
	chainhttp "chain/net/http"
)

func Handler() chainhttp.Handler {
	h := chainhttp.PatServeMux{PatternServeMux: pat.New()}
	h.AddFunc("POST", "/v3/applications/:applicationID/wallets", createWallet)
	h.AddFunc("POST", "/v3/wallets/:walletID/buckets", createBucket)
	h.AddFunc("POST", "/v3/wallets/:walletID/assets", createAsset)
	h.AddFunc("POST", "/v3/assets/:assetID/issue", issueAsset)
	h.AddFunc("POST", "/v3/assets/transfer", walletBuild)
	h.AddFunc("POST", "/v3/wallets/transact/finalize", walletFinalize)
	return h
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
	var outs []asset.Output
	err := json.NewDecoder(req.Body).Decode(&outs)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	assetID := req.URL.Query().Get(":assetID")

	template, err := asset.Issue(ctx, assetID, outs)
	if err != nil {
		// w.WriteHeader(httperror.Status(err)) // i wish
		w.WriteHeader(400) // not really
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"template": template,
	})

}

// /v3/assets/transfer
func walletBuild(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	panic("TODO")
}

// /v3/wallets/transact/finalize
func walletFinalize(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	// TODO(kr): validate

	var tpl asset.Tx
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

	err = appdb.Commit(ctx, tx)
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
