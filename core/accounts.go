package core

import (
	"sync"
	"time"

	"golang.org/x/net/context"

	"chain/core/account"
	"chain/metrics"
)

// POST /create-account
func createAccount(ctx context.Context, ins []struct {
	XPubs  []string
	Quorum int
	Tags   map[string]interface{}

	// ClientToken is the application's unique token for the account. Every account
	// should have a unique client token. The client token is used to ensure
	// idempotency of create account requests. Duplicate create account requests
	// with the same client_token will only create one account.
	ClientToken *string `json:"client_token"`
}) interface{} {
	defer metrics.RecordElapsed(time.Now())

	responses := make([]interface{}, len(ins))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			defer wg.Done()
			acc, err := account.Create(ctx, ins[i].XPubs, ins[i].Quorum, ins[i].Tags, ins[i].ClientToken)
			if err != nil {
				logHTTPError(ctx, err)
				responses[i], _ = errInfo(err)
			} else {
				responses[i] = acc
			}
		}(i)
	}

	wg.Wait()
	return responses
}

// POST /set-account-tags
func setAccountTags(ctx context.Context, in struct {
	AccountID string `json:"account_id"`
	Tags      map[string]interface{}
}) (interface{}, error) {
	return account.SetTags(ctx, in.AccountID, in.Tags)
}

// POST /archive-account
func archiveAccount(ctx context.Context, in struct {
	AccountID string `json:"account_id"`
}) error {
	return account.Archive(ctx, in.AccountID)
}
