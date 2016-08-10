package core

import (
	"sync"
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
	Tags       map[string]interface{} `json:"tags"`
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

// POST /update-asset
func setAssetTags(ctx context.Context, in struct {
	AssetID string `json:"asset_id"`
	Tags    map[string]interface{}
}) (interface{}, error) {
	var decodedAssetID bc.AssetID
	err := decodedAssetID.UnmarshalText([]byte(in.AssetID))
	if err != nil {
		return nil, errors.WithDetailf(httpjson.ErrBadRequest, "%q is an invalid asset ID", in.AssetID)
	}
	return asset.SetTags(ctx, decodedAssetID, in.Tags)
}

type assetResponseOrError struct {
	*assetResponse
	*detailedError
}

// POST /create-asset
func (a *api) createAsset(ctx context.Context, ins []struct {
	XPubs      []string
	Quorum     int
	Definition map[string]interface{}
	Tags       map[string]interface{}

	// ClientToken is the application's unique token for the asset. Every asset
	// should have a unique client token. The client token is used to ensure
	// idempotency of create asset requests. Duplicate create asset requests
	// with the same client_token will only create one asset.
	ClientToken *string `json:"client_token"`
}) ([]assetResponseOrError, error) {
	defer metrics.RecordElapsed(time.Now())

	genesis, err := a.store.GetBlock(ctx, 1)
	if err != nil {
		return nil, err
	}

	responses := make([]assetResponseOrError, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			defer wg.Done()
			asset, err := asset.Define(
				ctx,
				ins[i].XPubs,
				ins[i].Quorum,
				ins[i].Definition,
				genesis.Hash(),
				ins[i].Tags,
				ins[i].ClientToken,
			)
			if err != nil {
				logHTTPError(ctx, err)
				res, _ := errInfo(err)
				responses[i] = assetResponseOrError{detailedError: &res}
			} else {
				responses[i] = assetResponseOrError{
					assetResponse: &assetResponse{
						ID:         asset.AssetID,
						XPubs:      asset.Signer.XPubs,
						Quorum:     asset.Signer.Quorum,
						Definition: asset.Definition,
						Tags:       asset.Tags,
					},
				}
			}
		}(i)
	}

	wg.Wait()
	return responses, nil
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
