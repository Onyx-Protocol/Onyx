package core

import (
	"context"
	"encoding/json"
	"sync"

	"chain/core/query"
	"chain/core/signers"
	"chain/crypto/ed25519/chainkd"
	chainjson "chain/encoding/json"
	"chain/net/http/reqid"
)

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
			var keys []*query.AssetKey
			for _, xpub := range asset.Signer.XPubs {
				path := signers.Path(asset.Signer, signers.AssetKeySpace)
				var hexPath []chainjson.HexBytes
				for _, p := range path {
					hexPath = append(hexPath, p)
				}
				derived := xpub.Derive(path)
				keys = append(keys, &query.AssetKey{
					AssetPubkey:         derived[:],
					RootXPub:            xpub,
					AssetDerivationPath: hexPath,
				})
			}
			defRawMessage := json.RawMessage(asset.RawDefinition())
			if len(defRawMessage) == 0 {
				defRawMessage = json.RawMessage(`{}`)
			}
			tags, err := json.Marshal(asset.Tags)
			if err != nil {
				responses[i] = err
				return
			}
			tagsRawMessage := json.RawMessage(tags)

			aa := &query.AnnotatedAsset{
				ID:              asset.AssetID[:],
				VMVersion:       asset.VMVersion,
				IssuanceProgram: asset.IssuanceProgram,
				Keys:            keys,
				Quorum:          asset.Signer.Quorum,
				Definition:      &defRawMessage,
				RawDefinition:   chainjson.HexBytes(asset.RawDefinition()),
				Tags:            &tagsRawMessage,
				IsLocal:         true,
			}
			if asset.Alias != nil {
				aa.Alias = *asset.Alias
			}
			responses[i] = aa
		}(i)
	}

	wg.Wait()
	return responses, nil
}
