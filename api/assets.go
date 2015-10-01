package api

import (
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/net/http/httpjson"
)

// GET /v3/assets/:assetID/activity
func getAssetActivity(ctx context.Context, assetID string) (interface{}, error) {
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	activity, last, err := appdb.AssetActivity(ctx, assetID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":       last,
		"activities": httpjson.Array(activity),
	}
	return ret, nil
}
