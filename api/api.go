// Package api provides http handlers for all Chain operations.
package api

import (
	"bytes"
	"net/http"

	"golang.org/x/net/context"

	"github.com/tessr/pat"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/database/pg"
	"chain/encoding/json"
	"chain/fedchain/wire"
	chainhttp "chain/net/http"
)

func Handler() chainhttp.Handler {
	h := chainhttp.PatServeMux{PatternServeMux: pat.New()}
	h.AddFunc("POST", "/v3/applications/:appID/wallets", createWallet)
	h.AddFunc("POST", "/v3/wallets/:walletID/buckets", createBucket)
	h.AddFunc("POST", "/v3/applications/:appID/asset-groups", createAssetGroup)
	h.AddFunc("POST", "/v3/asset-groups/:groupID/assets", createAsset)
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
	err := readJSON(req.Body, &wReq)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	var keys []*appdb.Key
	for _, xpub := range wReq.XPubs {
		key, err := appdb.NewKey(xpub)
		if err != nil {
			writeHTTPError(ctx, w, err)
			return
		}
		keys = append(keys, key)
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}
	defer dbtx.Rollback()

	wID, err := appdb.CreateWallet(ctx, appID, wReq.Label, keys)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	err = dbtx.Commit()
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 201, map[string]interface{}{
		"wallet_id":           wID,
		"label":               wReq.Label,
		"block_chain":         "sandbox",
		"keys":                keys,
		"signatures_required": 1,
	})
}

// /v3/applications/:appID/asset-groups
func createAssetGroup(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	appID := req.URL.Query().Get(":appID")

	var agReq struct {
		Label string   `json:"label"`
		XPubs []string `json:"xpubs"`
	}
	err := readJSON(req.Body, &agReq)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	var keys []*appdb.Key
	for _, xpub := range agReq.XPubs {
		key, err := appdb.NewKey(xpub)
		if err != nil {
			writeHTTPError(ctx, w, err)
			return
		}
		keys = append(keys, key)
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}
	defer dbtx.Rollback()

	agID, err := appdb.CreateAssetGroup(ctx, appID, agReq.Label, keys)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	err = dbtx.Commit()
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 201, map[string]interface{}{
		"asset_group_id":      agID,
		"label":               agReq.Label,
		"block_chain":         "sandbox",
		"keys":                keys,
		"signatures_required": 1,
	})
}

// /v3/wallets/:walletID/buckets
func createBucket(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	walletID := req.URL.Query().Get(":walletID")

	var input struct {
		Label string `json:"label"`
	}
	err := readJSON(req.Body, &input)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	bucket, err := appdb.CreateBucket(ctx, walletID, input.Label)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 201, bucket)
}

// /v3/asset-groups/:groupID/assets
func createAsset(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	groupID := req.URL.Query().Get(":groupID")

	var input struct{ Label string }
	err := readJSON(req.Body, &input)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	asset, err := asset.Create(ctx, groupID, input.Label)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, http.StatusCreated, map[string]interface{}{
		"id":             asset.Hash.String(),
		"asset_group_id": asset.GroupID,
		"label":          asset.Label,
	})
}

// /v3/assets/:assetID/issue
func issueAsset(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	var outs []asset.Output
	err := readJSON(req.Body, &outs)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	assetID := req.URL.Query().Get(":assetID")
	template, err := asset.Issue(ctx, assetID, outs)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, map[string]interface{}{
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
	err := readJSON(req.Body, &tpl)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	tx := wire.NewMsgTx()
	err = tx.Deserialize(bytes.NewReader(tpl.Unsigned))
	if err != nil {
		writeHTTPError(ctx, w, err)
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
		writeHTTPError(ctx, w, err)
		return
	}
	defer dbtx.Rollback()

	err = appdb.Commit(ctx, tx)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	err = dbtx.Commit()
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	var buf bytes.Buffer
	tx.Serialize(&buf)

	writeJSON(ctx, w, 200, map[string]interface{}{
		"transaction_id":  tx.TxSha().String(),
		"raw_transaction": json.HexBytes(buf.Bytes()),
	})
}

// POST /v3/users
func createUser(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	var in struct {
		Email    string
		Password string
	}

	err := readJSON(req.Body, &in)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	user, err := appdb.CreateUser(ctx, in.Email, in.Password)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, user)
}
