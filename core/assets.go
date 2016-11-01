package core

import (
	"context"
	"sync"

	"chain/core/signers"
	"chain/encoding/json"
	"chain/net/http/reqid"
)

// This type enforces JSON field ordering in API output.
type assetResponse struct {
	ID              interface{} `json:"id"`
	Alias           *string     `json:"alias"`
	IssuanceProgram interface{} `json:"issuance_program"`
	Keys            interface{} `json:"keys"`
	Quorum          interface{} `json:"quorum"`
	Definition      interface{} `json:"definition"`
	Tags            interface{} `json:"tags"`
	IsLocal         interface{} `json:"is_local"`
}

type assetKey struct {
	RootXPub            interface{} `json:"root_xpub"`
	AssetPubkey         interface{} `json:"asset_pubkey"`
	AssetDerivationPath interface{} `json:"asset_derivation_path"`
}

// POST /create-asset
func (h *Handler) createAsset(ctx context.Context, ins []struct {
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
}) ([]interface{}, error) {
	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := range responses {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			asset, err := h.Assets.Define(
				subctx,
				ins[i].RootXPubs,
				ins[i].Quorum,
				ins[i].Definition,
				ins[i].Alias,
				ins[i].Tags,
				ins[i].ClientToken,
			)
			if err != nil {
				responses[i] = err
				return
			}
			var keys []assetKey
			for _, xpub := range asset.Signer.XPubs {
				path := signers.Path(asset.Signer, signers.AssetKeySpace)
				derived := xpub.Derive(path)
				keys = append(keys, assetKey{
					AssetPubkey:         json.HexBytes(derived[:]),
					RootXPub:            xpub,
					AssetDerivationPath: path,
				})
			}
			responses[i] = &assetResponse{
				ID:              asset.AssetID,
				Alias:           asset.Alias,
				IssuanceProgram: asset.IssuanceProgram,
				Keys:            keys,
				Quorum:          asset.Signer.Quorum,
				Definition:      asset.Definition,
				Tags:            asset.Tags,
				IsLocal:         "yes",
			}
		}(i)
	}

	wg.Wait()
	return responses, nil
}
