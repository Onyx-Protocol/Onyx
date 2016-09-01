package core

import (
	"context"
	"sync"
	"time"

	"chain/core/asset"
	"chain/errors"
	"chain/metrics"
	"chain/net/http/httpjson"
	"chain/protocol/bc"
)

type assetResponse struct {
	ID              bc.AssetID             `json:"id"`
	IssuanceProgram []byte                 `json:"issuance_program"`
	Definition      map[string]interface{} `json:"definition"`
	Tags            map[string]interface{} `json:"tags"`
}

// POST /update-asset
func setAssetTags(ctx context.Context, in struct {
	AssetID string `json:"asset_id"`
	Alias   string `json:"alias"`
	Tags    map[string]interface{}
}) (interface{}, error) {
	var decodedAssetID bc.AssetID
	if in.AssetID != "" {
		err := decodedAssetID.UnmarshalText([]byte(in.AssetID))
		if err != nil {
			return nil, errors.WithDetailf(httpjson.ErrBadRequest, "%q is an invalid asset ID", in.AssetID)
		}

		if in.Alias != "" {
			return nil, errors.Wrap(httpjson.ErrBadRequest, "cannot supply both asset_id and alias")
		}
	}

	if in.AssetID == "" && in.Alias == "" {
		return nil, errors.Wrap(httpjson.ErrBadRequest, "must supply either asset_id or alias")
	}

	return asset.SetTags(ctx, decodedAssetID, in.Alias, in.Tags)
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
	Alias      string
	Tags       map[string]interface{}

	// ClientToken is the application's unique token for the asset. Every asset
	// should have a unique client token. The client token is used to ensure
	// idempotency of create asset requests. Duplicate create asset requests
	// with the same client_token will only create one asset.
	ClientToken *string `json:"client_token"`
}) ([]assetResponseOrError, error) {
	defer metrics.RecordElapsed(time.Now())

	initialBlock, err := a.c.GetBlock(ctx, 1)
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
				initialBlock.Hash(),
				ins[i].Alias,
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
						ID:              asset.AssetID,
						IssuanceProgram: asset.IssuanceProgram,
						Definition:      asset.Definition,
						Tags:            asset.Tags,
					},
				}
			}
		}(i)
	}

	wg.Wait()
	return responses, nil
}

// POST /archive-asset
func archiveAsset(ctx context.Context, in struct {
	AssetID bc.AssetID `json:"asset_id"`
	Alias   string     `json:"alias"`
}) error {
	if (in.AssetID != bc.AssetID{} && in.Alias != "") {
		return errors.Wrap(httpjson.ErrBadRequest, "cannot supply both asset_id and alias")
	}

	if (in.AssetID == bc.AssetID{} && in.Alias == "") {
		return errors.Wrap(httpjson.ErrBadRequest, "must supply either asset_id or alias")
	}
	return asset.Archive(ctx, in.AssetID, in.Alias)
}
