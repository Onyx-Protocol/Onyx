// Package api provides http handlers for all Chain operations.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

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
	h.AddFunc("POST", "/v3/applications/:appID/wallets", createWallet)
	h.AddFunc("POST", "/v3/wallets/:walletID/buckets", createBucket)
	h.AddFunc("POST", "/v3/wallets/:walletID/assets", createAsset)
	h.AddFunc("POST", "/v3/assets/:assetID/issue", issueAsset)
	h.AddFunc("POST", "/v3/assets/transfer", walletBuild)
	h.AddFunc("POST", "/v3/wallets/transact/finalize", walletFinalize)
	h.AddFunc("POST", "/v3/users", createUser)
	return h
}

// /v3/applications/:appID/wallets
func createWallet(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	appID := req.URL.Query().Get(":appID")

	var wReq struct {
		Label string   `json:"label"`
		XPubs []string `json:"xpubs"`
	}
	err := json.NewDecoder(req.Body).Decode(&wReq)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	var keys []*appdb.Key
	for _, xpub := range wReq.XPubs {
		key, err := appdb.NewKey(xpub)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		keys = append(keys, key)
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer dbtx.Rollback()

	wID, err := appdb.CreateWallet(ctx, appID, wReq.Label, keys)
	if err != nil {
		// TODO(kr): distinguish between user and server error
		w.WriteHeader(400)
		return
	}

	err = dbtx.Commit()
	if err != nil {
		w.WriteHeader(500)
		return
	}

	writeJSON(w, 201, map[string]interface{}{
		"wallet_id":           wID,
		"label":               wReq.Label,
		"block_chain":         "sandbox",
		"keys":                keys,
		"signatures_required": 1,
	})
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

// POST /v3/users
func createUser(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	var in struct {
		Email    string
		Password string
	}

	err := json.NewDecoder(req.Body).Decode(&in)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	// TODO(jeffomatic) - these validations could be moved into CreateUser. This
	// will be easier once we create an app-specific error interface that had
	// pre-defined HTTP status codes and error messages.

	if len(in.Email) < 1 || 255 < len(in.Email) ||
		!strings.Contains(in.Email, "@") ||
		len(in.Password) < 6 || 255 < len(in.Password) {
		w.WriteHeader(400)
		return
	}

	user, err := appdb.CreateUser(ctx, in.Email, in.Password)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	writeJSON(w, 200, user)
}
