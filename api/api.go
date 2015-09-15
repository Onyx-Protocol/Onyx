// Package api provides http handlers for all Chain operations.
package api

import (
	"bytes"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/database/pg"
	"chain/encoding/json"
	"chain/errors"
	chainhttp "chain/net/http"
	"chain/net/http/authn"
	"chain/net/http/pat"
)

const (
	sessionTokenLifetime = 2 * 7 * 24 * time.Hour
	defActivityPageSize  = 50
)

func Handler() chainhttp.Handler {
	h := chainhttp.PatServeMux{PatternServeMux: pat.New()}
	h.AddFunc("GET", "/v3/applications", tokenAuthn(listApplications))
	h.AddFunc("POST", "/v3/applications", tokenAuthn(createApplication))
	h.AddFunc("GET", "/v3/applications/:appID", getApplication)
	h.AddFunc("PUT", "/v3/applications/:appID", updateApplication)
	h.AddFunc("GET", "/v3/applications/:appID/members", listMembers)
	h.AddFunc("POST", "/v3/applications/:appID/members", addMember)
	h.AddFunc("PUT", "/v3/applications/:appID/members/:userID", updateMember)
	h.AddFunc("DELETE", "/v3/applications/:appID/members/:userID", removeMember)
	h.AddFunc("GET", "/v3/applications/:appID/wallets", listWallets)
	h.AddFunc("POST", "/v3/applications/:appID/wallets", createWallet)
	h.AddFunc("GET", "/v3/wallets/:walletID", getWallet)
	h.AddFunc("GET", "/v3/wallets/:walletID/buckets", listBuckets)
	h.AddFunc("POST", "/v3/wallets/:walletID/buckets", createBucket)
	h.AddFunc("GET", "/v3/buckets/:bucketID/balance", getBucketBalance)
	h.AddFunc("GET", "/v3/wallets/:walletID/balance", getWalletBalance)
	h.AddFunc("GET", "/v3/wallets/:walletID/activity", getWalletActivity)
	h.AddFunc("POST", "/v3/applications/:appID/asset-groups", createAssetGroup)
	h.AddFunc("POST", "/v3/asset-groups/:groupID/assets", createAsset)
	h.AddFunc("POST", "/v3/buckets/:bucketID/addresses", createAddr)
	h.AddFunc("POST", "/v3/assets/:assetID/issue", issueAsset)
	h.AddFunc("POST", "/v3/assets/transfer", transferAssets)
	h.AddFunc("POST", "/v3/wallets/transact/finalize", walletFinalize)
	h.AddFunc("POST", "/v3/users", createUser)
	h.AddFunc("GET", "/v3/user", tokenAuthn(getAuthdUser))
	h.AddFunc("POST", "/v3/login", userCredsAuthn(login))
	h.AddFunc("GET", "/v3/authcheck", tokenAuthn(authcheck))
	h.AddFunc("GET", "/v3/api-tokens", tokenAuthn(listAPITokens))
	h.AddFunc("POST", "/v3/api-tokens", tokenAuthn(createAPIToken))
	h.AddFunc("DELETE", "/v3/api-tokens/:tokenID", deleteAPIToken)
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
	for i, xpub := range wReq.XPubs {
		key, err := appdb.NewKey(xpub)
		if err != nil {
			err = errors.Wrap(appdb.ErrBadXPub, err.Error())
			writeHTTPError(ctx, w, errors.WithDetailf(err, "xpub %d", i))
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
		"id":                  wID,
		"label":               wReq.Label,
		"block_chain":         "sandbox",
		"keys":                keys,
		"signatures_required": 1,
	})
}

// GET /v3/wallets/:walletID
func getWallet(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get(":walletID")
	wal, err := appdb.GetWallet(ctx, id)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, wal)
}

// /v3/wallets/:walletID/balance
func getWalletBalance(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	wID := req.URL.Query().Get(":walletID")
	bals, err := appdb.WalletBalance(ctx, wID)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, bals)
}

// GET /v3/applications/:appID/wallets
func listWallets(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	aid := req.URL.Query().Get(":appID")
	wallets, err := appdb.ListWallets(ctx, aid)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, wallets)
}

// /v3/wallets/:walletID/activity
func getWalletActivity(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	wID := req.URL.Query().Get(":walletID")
	prev := req.Header.Get("Range-After")

	limit := defActivityPageSize
	if lstr := req.Header.Get("Limit"); lstr != "" {
		var err error
		limit, err = strconv.Atoi(lstr)
		if err != nil {
			err = errors.Wrap(ErrBadReqHeader, err.Error())
			writeHTTPError(ctx, w, errors.WithDetail(err, "limit header"))
			return
		}
	}

	activity, last, err := appdb.WalletActivity(ctx, wID, prev, limit)

	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, map[string]interface{}{
		"last":       last,
		"activities": activity,
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
		"id":                  agID,
		"label":               agReq.Label,
		"block_chain":         "sandbox",
		"keys":                keys,
		"signatures_required": 1,
	})
}

// GET /v3/wallets/:walletID/buckets
func listBuckets(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get(":walletID")
	buckets, err := appdb.ListBuckets(ctx, id)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}
	writeJSON(ctx, w, 200, buckets)
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

// /v3/buckets/:bucketID/balance
func getBucketBalance(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	bID := req.URL.Query().Get(":bucketID")
	bals, err := appdb.BucketBalance(ctx, bID)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, bals)
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
func transferAssets(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	var x struct {
		Inputs  []asset.TransferInput
		Outputs []asset.Output
	}
	err := readJSON(req.Body, &x)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}
	defer dbtx.Rollback()

	template, err := asset.Transfer(ctx, x.Inputs, x.Outputs)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	err = dbtx.Commit()
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	writeJSON(ctx, w, 200, map[string]interface{}{
		"template": template,
	})
}

// /v3/wallets/transact/finalize
func walletFinalize(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	// TODO(kr): validate

	tpl := new(asset.Tx)
	err := readJSON(req.Body, tpl)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}
	defer dbtx.Rollback()

	tx, err := asset.FinalizeTx(ctx, tpl)
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

// POST /v3/login
func login(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	uid := authn.GetAuthID(ctx)
	expiresAt := time.Now().UTC().Add(sessionTokenLifetime)
	t, err := appdb.CreateAuthToken(ctx, uid, "session", &expiresAt)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}
	writeJSON(ctx, w, 200, t)
}

// GET /v3/user
func getAuthdUser(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	uid := authn.GetAuthID(ctx)
	u, err := appdb.GetUser(ctx, uid)
	if err != nil {
		writeHTTPError(ctx, w, err)
		return
	}
	writeJSON(ctx, w, 200, u)
}

// GET /v3/authcheck
func authcheck(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	writeJSON(ctx, w, 200, map[string]string{"message": "ok"})
}

// optionalTime returns a pointer to t or nil, if t is zero.
// It is helpful for JSON structs with omitempty.
func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
