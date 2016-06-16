package core

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/explorer"
	"chain/cos/bc"
	"chain/errors"
	"chain/net/http/httpjson"
)

func (a *api) getBlockSummary(ctx context.Context, hash string) (*explorer.BlockSummary, error) {
	return explorer.GetBlockSummary(ctx, a.store, hash)
}

func (a *api) getTx(ctx context.Context, txHashStr string) (*explorer.Tx, error) {
	return explorer.GetTx(ctx, a.store, a.pool, txHashStr)
}

func (a *api) getAsset(ctx context.Context, assetID string) (*explorer.Asset, error) {
	var decodedAssetID bc.AssetID
	err := decodedAssetID.UnmarshalText([]byte(assetID))
	if err != nil {
		return nil, errors.WithDetailf(httpjson.ErrBadRequest, "%q is an invalid asset ID", assetID)
	}
	return explorer.GetAsset(ctx, decodedAssetID)
}

func (a *api) listBlocks(ctx context.Context) (interface{}, error) {
	prev, limit, err := getPageData(ctx, 50)
	if err != nil {
		return nil, err
	}

	list, last, err := explorer.ListBlocks(ctx, a.store, prev, limit)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"blocks": httpjson.Array(list),
		"last":   last,
	}, nil
}

// EXPERIMENTAL(jeffomatic), implemented for R3 demo. Before baking it into the
// public API, we should decide whether this style of API querying is desirable.
func (a *api) getExplorerAssets(ctx context.Context, req struct {
	AssetIDs []bc.AssetID `json:"asset_ids"`
}) (interface{}, error) {
	assets, err := explorer.GetAssets(ctx, req.AssetIDs)
	if err != nil {
		return nil, err
	}
	var res []*explorer.Asset
	for _, a := range assets {
		res = append(res, a)
	}
	return res, nil
}

func (a *api) listExplorerUTXOsByAsset(ctx context.Context, assetID string) (interface{}, error) {
	prev, limit, err := getPageData(ctx, 50)
	if err != nil {
		return nil, err
	}

	h, err := bc.ParseHash(assetID)
	if err != nil {
		return nil, errors.WithDetailf(httpjson.ErrBadRequest, "invalid asset ID: %q", assetID)
	}

	var ts time.Time
	qvals := httpjson.Request(ctx).URL.Query()
	if timestamps, ok := qvals["timestamp"]; ok {
		timestamp := timestamps[0]
		ts, err = parseTime(timestamp)
		if err != nil {
			return nil, errors.WithDetailf(httpjson.ErrBadRequest, "invalid timestamp: %q", timestamp)
		}
	}
	list, last, err := explorer.ListHistoricalOutputsByAsset(ctx, bc.AssetID(h), ts, prev, limit)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"utxos": httpjson.Array(list),
		"last":  last,
	}, nil
}
