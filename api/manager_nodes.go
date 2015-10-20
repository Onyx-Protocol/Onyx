package api

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/database/pg"
	"chain/fedchain-sandbox/hdkey"
	"chain/metrics"
	"chain/net/http/httpjson"
)

// POST /v3/projects/:projID/manager-nodes
func createManagerNode(ctx context.Context, projID string, wReq *asset.CreateNodeReq) (interface{}, error) {
	if err := projectAuthz(ctx, projID); err != nil {
		return nil, err
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	managerNode, err := asset.CreateNode(ctx, asset.ManagerNode, projID, wReq)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, err
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
func deleteManagerNode(ctx context.Context, mnodeID string) error {
	if err := managerAuthz(ctx, mnodeID); err != nil {
		return err
	}
	return appdb.DeleteManagerNode(ctx, mnodeID)
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

	activity, last, err := appdb.ManagerNodeActivity(ctx, mnID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":       last,
		"activities": httpjson.Array(activity),
	}
	return ret, nil
}

// GET /v3/manager-nodes/:mnodeID/transactions/:txID
func managerNodeTxActivity(ctx context.Context, mnodeID, txID string) (interface{}, error) {
	if err := managerAuthz(ctx, mnodeID); err != nil {
		return nil, err
	}
	return appdb.ManagerNodeTxActivity(ctx, mnodeID, txID)
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

	balances, last, err := appdb.ManagerNodeBalance(ctx, managerNodeID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":     last,
		"balances": httpjson.Array(balances),
	}
	return ret, nil
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
func createAccount(ctx context.Context, managerNodeID string, in struct{ Label string }) (*appdb.Account, error) {
	defer metrics.RecordElapsed(time.Now())
	if err := managerAuthz(ctx, managerNodeID); err != nil {
		return nil, err
	}
	return appdb.CreateAccount(ctx, managerNodeID, in.Label)
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

	activity, last, err := appdb.AccountActivity(ctx, bid, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":       last,
		"activities": httpjson.Array(activity),
	}
	return ret, nil
}

// GET /v3/accounts/:accountID/balance
func accountBalance(ctx context.Context, accountID string) (interface{}, error) {
	if err := accountAuthz(ctx, accountID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defBalancePageSize)
	if err != nil {
		return nil, err
	}

	balances, last, err := appdb.AccountBalance(ctx, accountID, prev, limit)
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
func deleteAccount(ctx context.Context, accountID string) error {
	if err := accountAuthz(ctx, accountID); err != nil {
		return err
	}
	return appdb.DeleteAccount(ctx, accountID)
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
		IsChange:  false,
	}
	err := asset.CreateAddress(ctx, addr, true)
	if err != nil {
		return nil, err
	}

	signers := hdkey.Derive(addr.Keys, appdb.ReceiverPath(addr, addr.Index))
	ret := map[string]interface{}{
		"address":             addr.Address,
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
