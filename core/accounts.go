package core

import (
	"context"
	"sync"

	"chain/core/signers"
	"chain/net/http/reqid"
)

// This type enforces JSON field ordering in API output.
type accountResponse struct {
	ID     interface{} `json:"id"`
	Alias  interface{} `json:"alias"`
	Keys   interface{} `json:"keys"`
	Quorum interface{} `json:"quorum"`
	Tags   interface{} `json:"tags"`
}

type accountKey struct {
	RootXPub              interface{} `json:"root_xpub"`
	AccountXPub           interface{} `json:"account_xpub"`
	AccountDerivationPath interface{} `json:"account_derivation_path"`
}

// POST /create-account
func (h *Handler) createAccount(ctx context.Context, ins []struct {
	RootXPubs []string `json:"root_xpubs"`
	Quorum    int
	Alias     string
	Tags      map[string]interface{}

	// ClientToken is the application's unique token for the account. Every account
	// should have a unique client token. The client token is used to ensure
	// idempotency of create account requests. Duplicate create account requests
	// with the same client_token will only create one account.
	ClientToken *string `json:"client_token"`
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
			var keys []accountKey
			for _, xpub := range acc.XPubs {
				keys = append(keys, accountKey{
					RootXPub:              xpub,
					AccountXPub:           xpub.Derive(path),
					AccountDerivationPath: path,
				})
			}
			responses[i] = &accountResponse{
				ID:     acc.ID,
				Alias:  acc.Alias,
				Keys:   keys,
				Quorum: acc.Quorum,
				Tags:   acc.Tags,
			}
		}(i)
	}

	wg.Wait()
	return responses
}
