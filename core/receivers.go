package core

import (
	"context"
	"sync"
	"time"

	"chain/net/http/reqid"
)

// POST /create-account-receiver
func (a *API) createAccountReceiver(ctx context.Context, ins []struct {
	AccountID    string    `json:"account_id"`
	AccountAlias string    `json:"account_alias"`
	ExpiresAt    time.Time `json:"expires_at"`
}) []interface{} {
	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			receiver, err := a.accounts.CreateReceiver(subctx, ins[i].AccountID, ins[i].AccountAlias, ins[i].ExpiresAt)
			if err != nil {
				responses[i] = err
			} else {
				responses[i] = receiver
			}
		}(i)
	}

	wg.Wait()
	return responses
}
