package core

import (
	"context"
	"sync"

	"chain/core/asset"
	"chain/crypto/ed25519/chainkd"
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

			a, err := h.Assets.Define(
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
			aa, err := asset.Annotated(a)
			if err != nil {
				responses[i] = err
				return
			}
			responses[i] = aa
		}(i)
	}

	wg.Wait()
	return responses, nil
}
