package core

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/asset"
	"chain/cos/bc"
	"chain/crypto/ed25519/hd25519"
	"chain/errors"
	"chain/metrics"
	"chain/net/http/httpjson"
)

type assetResponse struct {
	ID         bc.AssetID             `json:"id"`
	XPubs      []*hd25519.XPub        `json:"xpubs"`
	Quorum     int                    `json:"quorum"`
	Definition map[string]interface{} `json:"definition"`
}

// POST /list-assets
func (a *api) listAssets(ctx context.Context, in requestQuery) (result page, err error) {
	limit := defAccountPageSize

	assets, cursor, err := asset.List(ctx, in.Cursor, limit)
	if err != nil {
		return result, err
	}

	for _, asset := range assets {
		result.Items = append(result.Items, assetResponse{
			ID:         asset.AssetID,
			XPubs:      asset.Signer.XPubs,
			Quorum:     asset.Signer.Quorum,
			Definition: asset.Definition})
	}

	result.LastPage = len(assets) < limit
	result.Query.Cursor = cursor
	return result, nil
}

// POST /create-asset
func (a *api) createAsset(ctx context.Context, in struct {
	XPubs      []string
	Quorum     int
	Definition map[string]interface{}

	// ClientToken is the application's unique token for the asset. Every asset
	// should have a unique client token. The client token is used to ensure
	// idempotency of create asset requests. Duplicate create asset requests
	// with the same client_token will only create one asset.
	ClientToken *string `json:"client_token"`
}) (result assetResponse, err error) {
	defer metrics.RecordElapsed(time.Now())

	genesis, err := a.store.GetBlock(ctx, 1)
	if err != nil {
		return result, err
	}

	asset, err := asset.Define(ctx, in.XPubs, in.Quorum, in.Definition, genesis.Hash(), in.ClientToken)
	if err != nil {
		return result, err
	}

	result.ID = asset.AssetID
	result.XPubs = asset.Signer.XPubs
	result.Quorum = asset.Signer.Quorum
	result.Definition = asset.Definition
	return result, nil
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
