package core

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/asset"
	"chain/cos/bc"
	"chain/errors"
	"chain/metrics"
	"chain/net/http/httpjson"
)

// GET /v3/assets/:assetID
func (a *api) getAsset(ctx context.Context, assetID string) (*asset.Asset, error) {
	var aid bc.AssetID
	err := aid.UnmarshalText([]byte(assetID))
	if err != nil {
		return nil, errors.WithDetailf(httpjson.ErrBadRequest, "%q is an invalid asset ID", assetID)
	}
	asset, err := asset.Find(ctx, aid)
	if err != nil {
		return nil, err
	}
	return asset, nil
}

// GET /v3/assets
func (a *api) listAssets(ctx context.Context) (interface{}, error) {
	prev, limit, err := getPageData(ctx, defAssetPageSize)
	if err != nil {
		return nil, err
	}

	assets, last, err := asset.List(ctx, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":   last,
		"assets": httpjson.Array(assets),
	}
	return ret, nil
}

// POST /v3/assets
func (a *api) defineAsset(ctx context.Context, in struct {
	XPubs      []string
	Quorum     int
	Definition map[string]interface{}

	// ClientToken is the application's unique token for the asset. Every asset
	// should have a unique client token. The client token is used to ensure
	// idempotency of create asset requests. Duplicate create asset requests
	// with the same client_token will only create one asset.
	ClientToken *string `json:"client_token"`
}) (*asset.Asset, error) {
	defer metrics.RecordElapsed(time.Now())

	genesis, err := a.store.GetBlock(ctx, 1)
	if err != nil {
		return nil, err
	}

	return asset.Define(ctx, in.XPubs, in.Quorum, in.Definition, genesis.Hash(), in.ClientToken)
}

// DELETE /v3/assets/:assetID
// Idempotent
func archiveAsset(ctx context.Context, assetID string) error {
	var decodedAssetID bc.AssetID
	err := decodedAssetID.UnmarshalText([]byte(assetID))
	if err != nil {
		return errors.WithDetailf(httpjson.ErrBadRequest, "%q is an invalid asset ID", assetID)
	}
	return asset.Archive(ctx, decodedAssetID)
}
