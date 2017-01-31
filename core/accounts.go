package core

import (
	"context"
	"sync"

	"chain/core/account"
	"chain/crypto/ed25519/chainkd"
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
			aa, err := account.Annotated(acc)
			if err != nil {
				responses[i] = err
				return
			}
			responses[i] = aa
		}(i)
	}

	wg.Wait()
	return responses
}
