package core

import (
	"golang.org/x/net/context"

	"chain/core/explorer"
	"chain/cos/bc"
	"chain/errors"
	"chain/net/http/httpjson"
)

func (a *api) getBlockSummary(ctx context.Context, hash string) (*explorer.BlockSummary, error) {
	return a.explorer.GetBlockSummary(ctx, hash)
}

func (a *api) getTx(ctx context.Context, txHashStr string) (*explorer.Tx, error) {
	return a.explorer.GetTx(ctx, txHashStr)
}

func (a *api) getAsset(ctx context.Context, assetID string) (*explorer.Asset, error) {
	var decodedAssetID bc.AssetID
	err := decodedAssetID.UnmarshalText([]byte(assetID))
	if err != nil {
		return nil, errors.WithDetailf(httpjson.ErrBadRequest, "%q is an invalid asset ID", assetID)
	}
	return a.explorer.GetAsset(ctx, decodedAssetID)
}

func (a *api) listBlocks(ctx context.Context) (interface{}, error) {
	prev, limit, err := getPageData(ctx, 50)
	if err != nil {
		return nil, err
	}

	list, last, err := a.explorer.ListBlocks(ctx, prev, limit)
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
	assets, err := a.explorer.GetAssets(ctx, req.AssetIDs)
	if err != nil {
		return nil, err
	}
	var res []*explorer.Asset
	for _, a := range assets {
		res = append(res, a)
	}
	return res, nil
}
