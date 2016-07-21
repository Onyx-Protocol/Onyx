package core

import (
	"encoding/json"
	"strings"
	"time"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/asset"
	"chain/cos/bc"
	"chain/cos/hdkey"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/metrics"
	"chain/net/http/httpjson"
)

// POST /v3/projects/:projID/manager-nodes
func createManagerNode(ctx context.Context, projID string, req map[string]interface{}) (interface{}, error) {
	_, ok := req["keys"]
	isDeprecated := !ok

	bReq, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "trouble marshaling request")
	}

	var (
		managerNode interface{}
		cnReq       asset.CreateNodeReq
	)

	if isDeprecated {
		var depReq asset.DeprecatedCreateNodeReq
		err = json.Unmarshal(bReq, &depReq)
		if err != nil {
			return nil, errors.Wrap(err, "invalid account manager creation request")
		}

		for _, xp := range depReq.XPubs {
			key := &asset.CreateNodeKeySpec{Type: "service", XPub: xp}
			cnReq.Keys = append(cnReq.Keys, key)
		}

		if depReq.GenerateKey {
			key := &asset.CreateNodeKeySpec{Type: "service", Generate: true}
			cnReq.Keys = append(cnReq.Keys, key)
		}

		cnReq.SigsRequired = 1
		cnReq.Label = depReq.Label
	} else {
		err = json.Unmarshal(bReq, &cnReq)
		if err != nil {
			return nil, errors.Wrap(err, "invalid account manager creation request")
		}
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "begin tx")
	}
	defer dbtx.Rollback(ctx)

	managerNode, err = asset.CreateNode(ctx, asset.ManagerNode, projID, &cnReq)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "commit tx")
	}

	return managerNode, nil
}

// PUT /v3/manager-nodes/:mnodeID
func updateManagerNode(ctx context.Context, mnodeID string, in struct{ Label *string }) error {
	if err := managerAuthz(ctx, mnodeID); err != nil {
		return err
	}
	return appdb.UpdateManagerNode(ctx, mnodeID, in.Label)
}

// DELETE /v3/manager-nodes/:mnodeID
// Idempotent
func archiveManagerNode(ctx context.Context, mnodeID string) error {
	if err := managerAuthz(ctx, mnodeID); errors.Root(err) == appdb.ErrArchived {
		// This manager node was already archived. Return success.
		return nil
	} else if err != nil {
		return err
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer dbtx.Rollback(ctx)

	err = appdb.ArchiveManagerNode(ctx, mnodeID)
	if err != nil {
		return err
	}

	return dbtx.Commit(ctx)
}

// GET /v3/projects/:projID/manager-nodes
func listManagerNodes(ctx context.Context, projID string) (interface{}, error) {
	return appdb.ListManagerNodes(ctx, projID)
}

// GET /v3/manager-nodes/:mnodeID
func getManagerNode(ctx context.Context, mnodeID string) (interface{}, error) {
	if err := managerAuthz(ctx, mnodeID); err != nil {
		return nil, err
	}
	return appdb.GetManagerNode(ctx, mnodeID)
}

// GET /v3/manager-nodes/:mnodeID/activity
func getManagerNodeActivity(ctx context.Context, mnID string) (interface{}, error) {
	if err := managerAuthz(ctx, mnID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	nodeTxs, last, err := appdb.ManagerTxs(ctx, mnID, prev, limit)
	if err != nil {
		return nil, err
	}

	activity, err := nodeTxsToActivity(nodeTxs)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":       last,
		"activities": httpjson.Array(activity),
	}
	return ret, nil
}

func managerNodeTx(ctx context.Context, mnodeID, txID string) (interface{}, error) {
	if err := managerAuthz(ctx, mnodeID); err != nil {
		return nil, err
	}
	return appdb.ManagerTx(ctx, mnodeID, txID)
}

// DEPRECATED - we will migrate the API to use managerNodeTx.
// GET /v3/manager-nodes/:mnodeID/transactions/:txID
func managerNodeTxActivity(ctx context.Context, mnodeID, txID string) (interface{}, error) {
	if err := managerAuthz(ctx, mnodeID); err != nil {
		return nil, err
	}
	nodeTx, err := appdb.ManagerTx(ctx, mnodeID, txID)
	if err != nil {
		return nil, err
	}
	return nodeTxToActivity(*nodeTx)
}

// GET /v3/manager-nodes/:mnodeID/transactions
func getManagerNodeTxs(ctx context.Context, mnID string) (interface{}, error) {
	if err := managerAuthz(ctx, mnID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	txs, last, err := appdb.ManagerTxs(ctx, mnID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":         last,
		"transactions": httpjson.Array(txs),
	}
	return ret, nil
}

// GET /v3/manager-nodes/:mnodeID/balance
func managerNodeBalance(ctx context.Context, managerNodeID string) (interface{}, error) {
	if err := managerAuthz(ctx, managerNodeID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defBalancePageSize)
	if err != nil {
		return nil, err
	}

	balances, last, err := appdb.AssetBalance(ctx, &appdb.AssetBalQuery{
		Owner:   appdb.OwnerManagerNode,
		OwnerID: managerNodeID,
		Prev:    prev,
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":     last,
		"balances": httpjson.Array(balances),
	}
	return ret, nil
}

// EXPERIMENTAL - implemented for Glitterco
func listAccountsWithAsset(ctx context.Context, mnodeID, assetID string) (interface{}, error) {
	if err := managerAuthz(ctx, mnodeID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defBalancePageSize)
	if err != nil {
		return nil, err
	}

	balances, last, err := appdb.AccountsWithAsset(ctx, mnodeID, assetID, prev, limit)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"balances": httpjson.Array(balances),
		"last":     last,
	}, nil
}

// GET /v3/manager-nodes/:mnodeID/accounts
func listAccounts(ctx context.Context, managerNodeID string) (interface{}, error) {
	if err := managerAuthz(ctx, managerNodeID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defAccountPageSize)
	if err != nil {
		return nil, err
	}

	accounts, last, err := appdb.ListAccounts(ctx, managerNodeID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":     last,
		"accounts": httpjson.Array(accounts),
	}
	return ret, nil
}

// POST /v3/manager-nodes/:mnodeID/accounts
// TODO(jackson): ClientToken should become required once all SDKs
// have been updated.
func createAccount(ctx context.Context, managerNodeID string, in struct {
	Label string
	Keys  []string

	// ClientToken is the application's unique token for the account. Every account
	// within a manager node should have a unique client token. The client token
	// is used to ensure idempotency of create account requests. Duplicate create
	// account requests within the same manager node with the same client_token will
	// only create one account.
	ClientToken *string `json:"client_token"`
}) (*appdb.Account, error) {
	defer metrics.RecordElapsed(time.Now())
	if err := managerAuthz(ctx, managerNodeID); err != nil {
		return nil, err
	}
	return appdb.CreateAccount(ctx, managerNodeID, in.Label, in.Keys, in.ClientToken)
}

// GET /v3/accounts/:accountID
func getAccount(ctx context.Context, accountID string) (interface{}, error) {
	if err := accountAuthz(ctx, accountID); err != nil {
		return nil, err
	}
	return appdb.GetAccount(ctx, accountID)
}

// GET /v3/accounts/:accountID/activity
func getAccountActivity(ctx context.Context, accountID string) (interface{}, error) {
	return getAccountTxsOrActivity(ctx, accountID, true)
}

// GET /v3/accounts/:accountID/transactions
func getAccountTxs(ctx context.Context, accountID string) (interface{}, error) {
	return getAccountTxsOrActivity(ctx, accountID, false)
}

func getAccountTxsOrActivity(ctx context.Context, accountID string, doActivity bool) (interface{}, error) {
	if err := accountAuthz(ctx, accountID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	// Defaults: startTime is the "beginning of time," endTime is now
	var startTime time.Time
	endTime := time.Now()

	qvals := httpjson.Request(ctx).URL.Query()
	if t, ok := qvals["start_time"]; ok {
		startTime, err = parseTime(t[0])
		if err != nil {
			return nil, errors.WithDetailf(httpjson.ErrBadRequest, "invalid timestamp: %q", t[0])
		}
	}
	if t, ok := qvals["end_time"]; ok {
		endTime, err = parseTime(t[0])
		if err != nil {
			return nil, errors.WithDetailf(httpjson.ErrBadRequest, "invalid timestamp: %q", t[0])
		}
	}

	txs, last, err := appdb.AccountTxs(ctx, accountID, startTime, endTime, prev, limit)
	if err != nil {
		return nil, err
	}

	var (
		key   string
		items interface{}
	)
	if doActivity {
		key = "activities"
		activity, err := nodeTxsToActivity(txs)
		if err != nil {
			return nil, err
		}
		items = activity
	} else {
		key = "transactions"
		items = txs
	}

	ret := map[string]interface{}{
		"last": last,
		key:    httpjson.Array(items),
	}
	return ret, nil
}

// GET /v3/accounts/:accountID/balance
func (a *api) accountBalance(ctx context.Context, accountID string) (interface{}, error) {
	var err error
	if err := accountAuthz(ctx, accountID); err != nil {
		return nil, err
	}

	query := &appdb.AssetBalQuery{
		Owner:   appdb.OwnerAccount,
		OwnerID: accountID,
	}

	qvals := httpjson.Request(ctx).URL.Query()
	if aidList, ok := qvals["asset_ids"]; ok {
		// EXPERIMENTAL - implemented for Glitterco
		//
		// Mode 1: filter by list of asset IDs
		// Asset IDs are serialized as a comma-separated list.
		query.AssetIDs = strings.Split(aidList[0], ",")
		if len(query.AssetIDs) == 0 {
			return map[string]interface{}{"balances": []string{}, "last": ""}, nil
		}
	} else {
		// Mode 2: return all assets, paginated by asset ID
		query.Prev, query.Limit, err = getPageData(ctx, defBalancePageSize)
		if err != nil {
			return nil, err
		}
	}

	var (
		amts     []bc.AssetAmount
		balances []*appdb.Balance
		last     string
	)
	if tsList, ok := qvals["timestamp"]; ok {
		timestamp, err := parseTime(tsList[0])
		if err != nil {
			return nil, errors.WithDetailf(httpjson.ErrBadRequest, "invalid timestamp: %q", timestamp)
		}

		var assetID *bc.AssetID
		if len(query.AssetIDs) > 0 {
			h, err := bc.ParseHash(query.AssetIDs[0])
			if err != nil {
				return nil, errors.WithDetailf(httpjson.ErrBadRequest, "invalid asset ID: %q", query.AssetIDs[0])
			}
			aid := bc.AssetID(h)
			assetID = &aid
		}

		amts, last, err = a.explorer.HistoricalBalancesByAccount(ctx, accountID, timestamp, assetID, query.Prev, query.Limit)
		if err != nil {
			return nil, err
		}

		for _, a := range amts {
			balances = append(balances, &appdb.Balance{
				AssetID:   a.AssetID,
				Confirmed: int64(a.Amount),
				Total:     int64(a.Amount),
			})
		}
	} else {
		balances, last, err = appdb.AssetBalance(ctx, query)
		if err != nil {
			return nil, err
		}

	}

	return map[string]interface{}{
		"last":     last,
		"balances": httpjson.Array(balances),
	}, nil
}

// PUT /v3/accounts/:accountID
func updateAccount(ctx context.Context, accountID string, in struct{ Label *string }) error {
	if err := accountAuthz(ctx, accountID); err != nil {
		return err
	}
	return appdb.UpdateAccount(ctx, accountID, in.Label)
}

// DELETE /v3/accounts/:accountID
// Idempotent
func archiveAccount(ctx context.Context, accountID string) error {
	if err := accountAuthz(ctx, accountID); errors.Root(err) == appdb.ErrArchived {
		// This account was already archived. Return success.
		return nil
	} else if err != nil {
		return err
	}
	return appdb.ArchiveAccount(ctx, accountID)
}

// /v3/accounts/:accountID/addresses
func createAddr(ctx context.Context, accountID string, in struct {
	Amount  uint64
	Expires time.Time
}) (interface{}, error) {
	if err := accountAuthz(ctx, accountID); err != nil {
		return nil, err
	}
	addr := &appdb.Address{
		AccountID: accountID,
		Amount:    in.Amount,
		Expires:   in.Expires,
	}
	err := appdb.CreateAddress(ctx, addr, true)
	if err != nil {
		return nil, err
	}

	signers := hdkey.Derive(addr.Keys, appdb.ReceiverPath(addr, addr.Index))
	ret := map[string]interface{}{
		"address":             chainjson.HexBytes(addr.PKScript), // deprecated
		"script":              chainjson.HexBytes(addr.PKScript),
		"signatures_required": addr.SigsRequired,
		"signers":             addrSigners(signers),
		"block_chain":         "sandbox",
		"created":             addr.Created.UTC(),
		"expires":             optionalTime(addr.Expires),
		"id":                  addr.ID,
		"index":               addr.Index[:],
	}
	return ret, nil
}

func addrSigners(signers []*hdkey.Key) (v []interface{}) {
	for _, s := range signers {
		v = append(v, map[string]interface{}{
			"pubkey":          s.Address.String(),
			"derivation_path": s.Path,
			"xpub":            s.Root.String(),
		})
	}
	return v
}

// optionalTime returns a pointer to t or nil, if t is zero.
// It is helpful for JSON structs with omitempty.
func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func listAccountUTXOs(ctx context.Context, accountID string, in struct {
	AssetIDs []bc.AssetID `json:"asset_ids"`
}) (interface{}, error) {
	if err := accountAuthz(ctx, accountID); err != nil {
		return nil, err
	}

	cursor, limit, err := getPageData(ctx, defGenericPageSize)
	if err != nil {
		return nil, err
	}

	utxos, last, err := appdb.ListAccountUTXOs(ctx, accountID, in.AssetIDs, cursor, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":         last,
		"transactions": httpjson.Array(utxos),
	}
	return ret, nil
}
