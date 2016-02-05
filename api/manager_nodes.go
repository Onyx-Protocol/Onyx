package api

import (
	"encoding/json"
	"strings"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/metrics"
	"chain/net/http/httpjson"
)

// POST /v3/projects/:projID/manager-nodes
func createManagerNode(ctx context.Context, projID string, req map[string]interface{}) (interface{}, error) {
	if err := projectAuthz(ctx, projID); err != nil {
		return nil, err
	}

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
			return nil, errors.Wrap(err, "invalid node creation request")
		}

		for _, xp := range depReq.XPubs {
			key := &asset.CreateNodeKeySpec{Type: "node", XPub: xp}
			cnReq.Keys = append(cnReq.Keys, key)
		}

		if depReq.GenerateKey {
			key := &asset.CreateNodeKeySpec{Type: "node", Generate: true}
			cnReq.Keys = append(cnReq.Keys, key)
		}

		cnReq.SigsRequired = 1
		cnReq.Label = depReq.Label
	} else {
		err = json.Unmarshal(bReq, &cnReq)
		if err != nil {
			return nil, errors.Wrap(err, "invalid node creation request")
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
func archiveManagerNode(ctx context.Context, mnodeID string) error {
	if err := managerAuthz(ctx, mnodeID); err != nil {
		return err
	}
	return appdb.ArchiveManagerNode(ctx, mnodeID)
}

// GET /v3/projects/:projID/manager-nodes
func listManagerNodes(ctx context.Context, projID string) (interface{}, error) {
	if err := projectAuthz(ctx, projID); err != nil {
		return nil, err
	}
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
func createAccount(ctx context.Context, managerNodeID string, in struct {
	Label string
	Keys  []string
}) (*appdb.Account, error) {
	defer metrics.RecordElapsed(time.Now())
	if err := managerAuthz(ctx, managerNodeID); err != nil {
		return nil, err
	}
	return appdb.CreateAccount(ctx, managerNodeID, in.Label, in.Keys)
}

// GET /v3/accounts/:accountID
func getAccount(ctx context.Context, accountID string) (interface{}, error) {
	if err := accountAuthz(ctx, accountID); err != nil {
		return nil, err
	}
	return appdb.GetAccount(ctx, accountID)
}

// GET /v3/accounts/:accountID/activity
func getAccountActivity(ctx context.Context, bid string) (interface{}, error) {
	if err := accountAuthz(ctx, bid); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	nodeTxs, last, err := appdb.AccountTxs(ctx, bid, prev, limit)
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

// GET /v3/accounts/:accountID/transactions
func getAccountTxs(ctx context.Context, bid string) (interface{}, error) {
	if err := accountAuthz(ctx, bid); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	txs, last, err := appdb.AccountTxs(ctx, bid, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":         last,
		"transactions": httpjson.Array(txs),
	}
	return ret, nil
}

// GET /v3/accounts/:accountID/balance
func accountBalance(ctx context.Context, accountID string) (interface{}, error) {
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

	balances, last, err := appdb.AssetBalance(ctx, query)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":     last,
		"balances": httpjson.Array(balances),
	}
	return ret, nil
}

// PUT /v3/accounts/:accountID
func updateAccount(ctx context.Context, accountID string, in struct{ Label *string }) error {
	if err := accountAuthz(ctx, accountID); err != nil {
		return err
	}
	return appdb.UpdateAccount(ctx, accountID, in.Label)
}

// DELETE /v3/accounts/:accountID
func archiveAccount(ctx context.Context, accountID string) error {
	if err := accountAuthz(ctx, accountID); err != nil {
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
