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

// POST /create-account
func (h *Handler) createAccount(ctx context.Context, ins []struct {
	RootXPubs []chainkd.XPub `json:"root_xpubs"`
	Quorum    int
	Alias     string
	Tags      map[string]interface{}

	// ClientToken is the application's unique token for the account. Every account
	// should have a unique client token. The client token is used to ensure
	// idempotency of create account requests. Duplicate create account requests
	// with the same client_token will only create one account.
	ClientToken string `json:"client_token"`
}) interface{} {
	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := range responses {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			acc, err := h.Accounts.Create(subctx, ins[i].RootXPubs, ins[i].Quorum, ins[i].Alias, ins[i].Tags, ins[i].ClientToken)
			if err != nil {
				responses[i] = err
				return
			}
			path := signers.Path(acc.Signer, signers.AccountKeySpace)
			var hexPath []chainjson.HexBytes
			for _, p := range path {
				hexPath = append(hexPath, p)
			}
			var keys []*query.AccountKey
			for _, xpub := range acc.XPubs {
				keys = append(keys, &query.AccountKey{
					RootXPub:              xpub,
					AccountXPub:           xpub.Derive(path),
					AccountDerivationPath: hexPath,
				})
			}

			tags, err := json.Marshal(acc.Tags)
			if err != nil {
				responses[i] = err
				return
			}
			rawTags := json.RawMessage(tags)

			responses[i] = &query.AnnotatedAccount{
				ID:     acc.ID,
				Alias:  acc.Alias,
				Keys:   keys,
				Quorum: acc.Quorum,
				Tags:   &rawTags,
			}
		}(i)
	}

	wg.Wait()
	return responses
}
