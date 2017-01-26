package core

import (
	"context"
	"sync"

	"chain/core/signers"
	"chain/crypto/ed25519/chainkd"
	"chain/encoding/json"
	"chain/net/http/reqid"
	"chain/protocol/bc"
)

// This type enforces JSON field ordering in API output.
type assetResponse struct {
	ID              bc.AssetID             `json:"id"`
	Alias           *string                `json:"alias"`
	VMVersion       uint64                 `json:"vm_version"`
	IssuanceProgram json.HexBytes          `json:"issuance_program"`
	Keys            []*assetKey            `json:"keys"`
	Quorum          int                    `json:"quorum"`
	Definition      map[string]interface{} `json:"definition"`
	RawDefinition   json.HexBytes          `json:"raw_definition"`
	Tags            map[string]interface{} `json:"tags"`
	IsLocal         string                 `json:"is_local"`
}

type assetKey struct {
	RootXPub            chainkd.XPub    `json:"root_xpub"`
	AssetPubkey         json.HexBytes   `json:"asset_pubkey"`
	AssetDerivationPath []json.HexBytes `json:"asset_derivation_path"`
}

// POST /create-asset
func (h *Handler) createAsset(ctx context.Context, ins []struct {
	Alias      string
	RootXPubs  []chainkd.XPub `json:"root_xpubs"`
	Quorum     int
	Definition map[string]interface{}
	Tags       map[string]interface{}

	// ClientToken is the application's unique token for the asset. Every asset
	// should have a unique client token. The client token is used to ensure
	// idempotency of create asset requests. Duplicate create asset requests
	// with the same client_token will only create one asset.
	ClientToken string `json:"client_token"`
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
			var keys []*assetKey
			for _, xpub := range asset.Signer.XPubs {
				path := signers.Path(asset.Signer, signers.AssetKeySpace)
				var hexPath []json.HexBytes
				for _, p := range path {
					hexPath = append(hexPath, p)
				}
				derived := xpub.Derive(path)
				keys = append(keys, &assetKey{
					AssetPubkey:         derived[:],
					RootXPub:            xpub,
					AssetDerivationPath: hexPath,
				})
			}
			parsedDef, _ := asset.Definition() // cannot fail because Assets.Define() would catch parsing issues
			responses[i] = &assetResponse{
				ID:              asset.AssetID,
				Alias:           asset.Alias,
				VMVersion:       asset.VMVersion,
				IssuanceProgram: asset.IssuanceProgram,
				Keys:            keys,
				Quorum:          asset.Signer.Quorum,
				Definition:      parsedDef,
				RawDefinition:   json.HexBytes(asset.RawDefinition()),
				Tags:            asset.Tags,
				IsLocal:         "yes",
			}
		}(i)
	}

	wg.Wait()
	return responses, nil
}
