package core

import (
	"context"
	"sync"
	"time"

	"chain/core/asset"
	"chain/core/signers"
	"chain/crypto/ed25519/hd25519"
	"chain/encoding/json"
	"chain/errors"
	"chain/metrics"
	"chain/net/http/httpjson"
	"chain/protocol/bc"
)

type (
	// This type enforces JSON field ordering in API output.
	assetResponse struct {
		ID              interface{} `json:"id"`
		Alias           *string     `json:"alias"`
		IssuanceProgram interface{} `json:"issuance_program"`
		Keys            interface{} `json:"keys"`
		Quorum          interface{} `json:"quorum"`
		Definition      interface{} `json:"definition"`
		Tags            interface{} `json:"tags"`
		Origin          interface{} `json:"origin"`
	}
	assetOrError struct {
		*assetResponse
		*detailedError
	}
)

type assetKey struct {
	AssetPubkey         interface{} `json:"account_pubkey"`
	RootXPub            interface{} `json:"root_xpub"`
	AssetDerivationPath interface{} `json:"asset_derivation_path"`
}

// POST /create-asset
func (a *api) createAsset(ctx context.Context, ins []struct {
	Alias      string
	RootXPubs  []string `json:"root_xpubs"`
	Quorum     int
	Definition map[string]interface{}
	Tags       map[string]interface{}

	// ClientToken is the application's unique token for the asset. Every asset
	// should have a unique client token. The client token is used to ensure
	// idempotency of create asset requests. Duplicate create asset requests
	// with the same client_token will only create one asset.
	ClientToken *string `json:"client_token"`
}) ([]assetOrError, error) {
	defer metrics.RecordElapsed(time.Now())

	initialBlock, err := a.c.GetBlock(ctx, 1)
	if err != nil {
		return nil, err
	}

	responses := make([]assetOrError, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			defer wg.Done()
			asset, err := asset.Define(
				ctx,
				ins[i].RootXPubs,
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
				responses[i] = assetOrError{detailedError: &res}
			} else {
				var keys []assetKey
				for _, xpub := range asset.Signer.XPubs {
					path := signers.Path(asset.Signer, signers.AssetKeySpace, nil)
					keys = append(keys, assetKey{
						AssetPubkey:         json.HexBytes(hd25519.PubBytes(xpub.Derive(path).Key)),
						RootXPub:            xpub,
						AssetDerivationPath: path,
					})
				}
				r := &assetResponse{
					ID:              asset.AssetID,
					Alias:           asset.Alias,
					IssuanceProgram: asset.IssuanceProgram,
					Keys:            keys,
					Quorum:          asset.Signer.Quorum,
					Definition:      asset.Definition,
					Tags:            asset.Tags,
					Origin:          "local",
				}
				responses[i] = assetOrError{assetResponse: r}
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
