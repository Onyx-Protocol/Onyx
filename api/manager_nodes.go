package api

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/metrics"
	"chain/net/http/httpjson"
)

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

// /v3/accounts/:accountID/addresses
func createAddr(ctx context.Context, bucketID string, in struct {
	Amount  uint64
	Expires time.Time
}) (interface{}, error) {
	addr := &appdb.Address{
		BucketID: bucketID,
		Amount:   in.Amount,
		Expires:  in.Expires,
		IsChange: false,
	}
	err := asset.CreateAddress(ctx, addr)
	if err != nil {
		return nil, err
	}

	signers := asset.Signers(addr.Keys, asset.ReceiverPath(addr))
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

func addrSigners(signers []*asset.DerivedKey) (v []interface{}) {
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
