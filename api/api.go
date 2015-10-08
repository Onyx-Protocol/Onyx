// Package api provides http handlers for all Chain operations.
package api

import (
	"bytes"
	"sync"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/encoding/json"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain-sandbox/wire"
	"chain/metrics"
	chainhttp "chain/net/http"
	"chain/net/http/authn"
	"chain/net/http/httpjson"
	"chain/net/http/pat"
)

const (
	sessionTokenLifetime = 2 * 7 * 24 * time.Hour
	defActivityPageSize  = 50
	defBucketPageSize    = 100
	defBalancePageSize   = 100
	defAssetPageSize     = 100
)

// Handler returns a handler that serves the Chain HTTP API. Param nouserSecret
// will be used as the password for routes starting with /nouser/.
func Handler(nouserSecret string) chainhttp.Handler {
	h := chainhttp.PatServeMux{PatternServeMux: pat.New()}

	pwHandler := httpjson.NewServeMux(writeHTTPError)
	pwHandler.HandleFunc("POST", "/v3/login", login)
	h.AddFunc("POST", "/v3/login", userCredsAuthn(pwHandler.ServeHTTPContext))

	nouserHandler := chainhttp.HandlerFunc(nouserAuthn(nouserSecret, nouserHandler()))
	h.Add("GET", "/nouser/", nouserHandler)
	h.Add("PUT", "/nouser/", nouserHandler)
	h.Add("POST", "/nouser/", nouserHandler)
	h.Add("DELETE", "/nouser/", nouserHandler)

	tokenHandler := chainhttp.HandlerFunc(tokenAuthn(tokenAuthedHandler()))
	h.Add("GET", "/", tokenHandler)
	h.Add("PUT", "/", tokenHandler)
	h.Add("POST", "/", tokenHandler)
	h.Add("DELETE", "/", tokenHandler)

	return h
}

func nouserHandler() chainhttp.HandlerFunc {
	h := httpjson.NewServeMux(writeHTTPError)

	// These routes must trust the client to enforce access control.
	// Think twice before adding something here.
	h.HandleFunc("GET", "/nouser/invitations/:invID", appdb.GetInvitation)
	h.HandleFunc("POST", "/nouser/invitations/:invID/create-user", createUserFromInvitation)
	h.HandleFunc("POST", "/nouser/invitations/:invID/add-existing", addMemberFromInvitation)
	h.HandleFunc("POST", "/nouser/password-reset/start", startPasswordReset)
	h.HandleFunc("POST", "/nouser/password-reset/finish", finishPasswordReset)

	return h.ServeHTTPContext
}

func tokenAuthedHandler() chainhttp.HandlerFunc {
	h := httpjson.NewServeMux(writeHTTPError)
	h.HandleFunc("GET", "/v3/projects", listApplications)
	h.HandleFunc("POST", "/v3/projects", createApplication)
	h.HandleFunc("GET", "/v3/projects/:projID", appdb.GetApplication)
	h.HandleFunc("PUT", "/v3/projects/:projID", updateApplication)
	h.HandleFunc("POST", "/v3/projects/:projID/invitations", createInvitation)
	h.HandleFunc("GET", "/v3/projects/:projID/members", appdb.ListMembers)
	h.HandleFunc("POST", "/v3/projects/:projID/members", addMember)
	h.HandleFunc("PUT", "/v3/projects/:projID/members/:userID", updateMember)
	h.HandleFunc("DELETE", "/v3/projects/:projID/members/:userID", appdb.RemoveMember)
	h.HandleFunc("GET", "/v3/projects/:projID/manager-nodes", appdb.ListWallets)
	h.HandleFunc("POST", "/v3/projects/:projID/manager-nodes", createWallet)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID", appdb.GetWallet)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID/accounts", listBuckets)
	h.HandleFunc("POST", "/v3/manager-nodes/:mnodeID/accounts", createBucket)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID/balance", walletBalance)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID/activity", getWalletActivity)
	h.HandleFunc("GET", "/v3/manager-nodes/:mnodeID/transactions/:txID", appdb.WalletTxActivity)
	h.HandleFunc("GET", "/v3/projects/:projID/issuer-nodes", appdb.ListAssetGroups)
	h.HandleFunc("POST", "/v3/projects/:projID/issuer-nodes", createAssetGroup)
	h.HandleFunc("GET", "/v3/issuer-nodes/:inodeID", appdb.GetAssetGroup)
	h.HandleFunc("GET", "/v3/issuer-nodes/:inodeID/assets", listAssets)
	h.HandleFunc("POST", "/v3/issuer-nodes/:inodeID/assets", createAsset)
	h.HandleFunc("GET", "/v3/accounts/:accountID/balance", bucketBalance)
	h.HandleFunc("GET", "/v3/accounts/:accountID/activity", getBucketActivity)
	h.HandleFunc("POST", "/v3/accounts/:accountID/addresses", createAddr)
	h.HandleFunc("GET", "/v3/assets/:assetID", appdb.GetAsset)
	h.HandleFunc("POST", "/v3/assets/:assetID/issue", issueAsset)
	h.HandleFunc("POST", "/v3/transact/build", build)
	h.HandleFunc("POST", "/v3/transact/transfer", transferAssets)
	h.HandleFunc("POST", "/v3/transact/transfer-batch", batchTransfer)
	h.HandleFunc("POST", "/v3/transact/trade", tradeAssets)
	h.HandleFunc("POST", "/v3/transact/finalize", walletFinalize)
	h.HandleFunc("POST", "/v3/transact/finalize-batch", batchFinalize)
	h.HandleFunc("POST", "/v3/transact/cancel-reservation", cancelReservation)
	h.HandleFunc("GET", "/v3/user", getAuthdUser)
	h.HandleFunc("POST", "/v3/user/email", updateUserEmail)
	h.HandleFunc("POST", "/v3/user/password", updateUserPassword)
	h.HandleFunc("GET", "/v3/authcheck", func() {})
	h.HandleFunc("GET", "/v3/api-tokens", listAPITokens)
	h.HandleFunc("POST", "/v3/api-tokens", createAPIToken)
	h.HandleFunc("DELETE", "/v3/api-tokens/:tokenID", appdb.DeleteAuthToken)
	return h.ServeHTTPContext
}

// POST /v3/projects/:projID/manager-nodes
func createWallet(ctx context.Context, appID string, wReq struct {
	Label string
	XPubs []string
}) (interface{}, error) {
	var keys []*hdkey.XKey
	for i, xpub := range wReq.XPubs {
		key, err := hdkey.NewXKey(xpub)
		if err != nil {
			err = errors.Wrap(appdb.ErrBadXPub, err.Error())
			return nil, errors.WithDetailf(err, "xpub %d", i)
		}
		keys = append(keys, key)
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	wID, err := appdb.CreateWallet(ctx, appID, wReq.Label, keys)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"id":                  wID,
		"label":               wReq.Label,
		"block_chain":         "sandbox",
		"keys":                keys,
		"signatures_required": 1,
	}
	return ret, nil
}

// GET /v3/manager-nodes/:mnodeID/activity
func getWalletActivity(ctx context.Context, wID string) (interface{}, error) {
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	activity, last, err := appdb.WalletActivity(ctx, wID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":       last,
		"activities": httpjson.Array(activity),
	}
	return ret, nil
}

// GET /v3/manager-nodes/:mnodeID/balance
func walletBalance(ctx context.Context, walletID string) (interface{}, error) {
	prev, limit, err := getPageData(ctx, defBalancePageSize)
	if err != nil {
		return nil, err
	}

	balances, last, err := appdb.WalletBalance(ctx, walletID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":     last,
		"balances": httpjson.Array(balances),
	}
	return ret, nil
}

// GET /v3/accounts/:accountID/balance
func bucketBalance(ctx context.Context, bucketID string) (interface{}, error) {
	prev, limit, err := getPageData(ctx, defBalancePageSize)
	if err != nil {
		return nil, err
	}

	balances, last, err := appdb.BucketBalance(ctx, bucketID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":     last,
		"balances": httpjson.Array(balances),
	}
	return ret, nil
}

// POST /v3/projects/:projID/issuer-nodes
func createAssetGroup(ctx context.Context, appID string, agReq struct {
	Label string
	XPubs []string
}) (interface{}, error) {
	var keys []*hdkey.XKey
	for _, xpub := range agReq.XPubs {
		key, err := hdkey.NewXKey(xpub)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	agID, err := appdb.CreateAssetGroup(ctx, appID, agReq.Label, keys)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"id":                  agID,
		"label":               agReq.Label,
		"block_chain":         "sandbox",
		"keys":                keys,
		"signatures_required": 1,
	}
	return ret, nil
}

// GET /v3/manager-nodes/:mnodeID/accounts
func listBuckets(ctx context.Context, walletID string) (interface{}, error) {
	prev, limit, err := getPageData(ctx, defBucketPageSize)
	if err != nil {
		return nil, err
	}

	buckets, last, err := appdb.ListBuckets(ctx, walletID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":     last,
		"accounts": httpjson.Array(buckets),
	}
	return ret, nil
}

// POST /v3/manager-nodes/:mnodeID/accounts
func createBucket(ctx context.Context, walletID string, in struct{ Label string }) (*appdb.Bucket, error) {
	defer metrics.RecordElapsed(time.Now())
	return appdb.CreateBucket(ctx, walletID, in.Label)
}

// GET /v3/accounts/:accountID/activity
func getBucketActivity(ctx context.Context, bid string) (interface{}, error) {
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	activity, last, err := appdb.BucketActivity(ctx, bid, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":       last,
		"activities": httpjson.Array(activity),
	}
	return ret, nil
}

// GET /v3/issuer-nodes/:inodeID/assets
func listAssets(ctx context.Context, groupID string) (interface{}, error) {
	prev, limit, err := getPageData(ctx, defAssetPageSize)
	if err != nil {
		return nil, err
	}

	assets, last, err := appdb.ListAssets(ctx, groupID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":   last,
		"assets": httpjson.Array(assets),
	}
	return ret, nil
}

// POST /v3/issuer-nodes/:inodeID/assets
func createAsset(ctx context.Context, groupID string, in struct{ Label string }) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	asset, err := asset.Create(ctx, groupID, in.Label)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"id":             asset.Hash.String(),
		"issuer_node_id": asset.GroupID,
		"label":          asset.Label,
	}
	return ret, nil
}

// POST /v3/assets/:assetID/issue
func issueAsset(ctx context.Context, assetID string, outs []asset.Output) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	template, err := asset.Issue(ctx, assetID, outs)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{"template": template}
	return ret, nil
}

type transferReq struct {
	Inputs  []utxodb.Input
	Outputs []asset.Output
}

// POST /v3/assets/transfer
func transferAssets(ctx context.Context, x transferReq) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	template, err := asset.Transfer(ctx, x.Inputs, x.Outputs)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{"template": template}
	return ret, nil
}

type buildReq struct {
	PrevTx  *asset.Tx `json:"previous_transaction"`
	Inputs  []utxodb.Input
	Outputs []asset.Output
	ResTime time.Duration `json:"reservation_duration"`
}

// POST /v3/transact/build
func build(ctx context.Context, buildReqs []buildReq) interface{} {
	defer metrics.RecordElapsed(time.Now())

	responses := make([]interface{}, len(buildReqs))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			defer wg.Done()

			dbtx, ctx, err := pg.Begin(ctx)
			if err != nil {
				responses[i], _ = errInfo(err)
				return
			}
			defer dbtx.Rollback()

			tpl, err := asset.Build(ctx, buildReqs[i].PrevTx, buildReqs[i].Inputs, buildReqs[i].Outputs, buildReqs[i].ResTime)
			if err != nil {
				responses[i], _ = errInfo(err)
				return
			}

			err = dbtx.Commit()
			if err != nil {
				responses[i], _ = errInfo(err)
				return
			}

			responses[i] = map[string]interface{}{"template": tpl}
		}(i)
	}

	wg.Wait()
	return responses
}

// POST /v3/assets/trade
func tradeAssets(ctx context.Context, x struct {
	PreviousTx *asset.Tx `json:"previous_transaction"`
	Inputs     []utxodb.Input
	Outputs    []asset.Output
}) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	template, err := asset.Trade(ctx, x.PreviousTx, x.Inputs, x.Outputs)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{"template": template}
	return ret, nil
}

// POST /v3/manager-nodes/transact/finalize
func walletFinalize(ctx context.Context, tpl *asset.Tx) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	// TODO(kr): validate

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	tx, err := asset.FinalizeTx(ctx, tpl)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	tx.Serialize(&buf)

	ret := map[string]interface{}{
		"transaction_id":  tx.TxSha().String(),
		"raw_transaction": json.HexBytes(buf.Bytes()),
	}
	return ret, nil
}

// POST /v3/assets/cancel-reservation
func cancelReservation(ctx context.Context, x struct {
	Transaction json.HexBytes
}) error {
	tx := wire.NewMsgTx()
	err := tx.Deserialize(bytes.NewReader(x.Transaction))
	if err != nil {
		return errors.Wrap(asset.ErrBadTxHex)
	}

	asset.CancelReservations(ctx, tx.OutPoints())
	return nil
}

// POST /v3/login
func login(ctx context.Context) (*appdb.AuthToken, error) {
	uid := authn.GetAuthID(ctx)
	expiresAt := time.Now().UTC().Add(sessionTokenLifetime)
	return appdb.CreateAuthToken(ctx, uid, "session", &expiresAt)
}

// GET /v3/user
func getAuthdUser(ctx context.Context) (*appdb.User, error) {
	uid := authn.GetAuthID(ctx)
	return appdb.GetUser(ctx, uid)
}

// POST /v3/user/email
func updateUserEmail(ctx context.Context, in struct{ Email, Password string }) error {
	uid := authn.GetAuthID(ctx)
	return appdb.UpdateUserEmail(ctx, uid, in.Password, in.Email)
}

// POST /v3/user/password
func updateUserPassword(ctx context.Context, in struct{ Current, New string }) error {
	uid := authn.GetAuthID(ctx)
	return appdb.UpdateUserPassword(ctx, uid, in.Current, in.New)
}

// POST /nouser/password-reset/start
func startPasswordReset(ctx context.Context, in struct{ Email string }) (interface{}, error) {
	secret, err := appdb.StartPasswordReset(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"secret": secret}, nil
}

// POST /nouser/password-reset/finish
func finishPasswordReset(ctx context.Context, in struct{ Email, Secret, Password string }) error {
	return appdb.FinishPasswordReset(ctx, in.Email, in.Secret, in.Password)
}

// optionalTime returns a pointer to t or nil, if t is zero.
// It is helpful for JSON structs with omitempty.
func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
