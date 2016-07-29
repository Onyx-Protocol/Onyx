package core

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/accounts"
	"chain/metrics"
	"chain/net/http/httpjson"
)

// GET /v3/accounts
func listAccounts(ctx context.Context) (interface{}, error) {
	prev, limit, err := getPageData(ctx, defAccountPageSize)
	if err != nil {
		return nil, err
	}

	accounts, last, err := accounts.List(ctx, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":     last,
		"accounts": httpjson.Array(accounts),
	}
	return ret, nil
}

// POST /v3/accounts
// TODO(jackson): ClientToken should become required once all SDKs
// have been updated.
func createAccount(ctx context.Context, in struct {
	XPubs  []string
	Quorum int

	// ClientToken is the application's unique token for the account. Every account
	// should have a unique client token. The client token s used to ensure
	// idempotency of create account requests. Duplicate create account requests
	// with the same client_token will only create one account.
	ClientToken *string `json:"client_token"`
}) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())

	return accounts.Create(ctx, in.XPubs, in.Quorum, in.ClientToken)
}

// GET /v3/accounts/:accountID
func getAccount(ctx context.Context, accountID string) (interface{}, error) {
	return accounts.Find(ctx, accountID)
}

// DELETE /v3/accounts/:accountID
// Idempotent
func archiveAccount(ctx context.Context, accountID string) error {
	return accounts.Archive(ctx, accountID)
}

// POST /v3/accounts/:accountID/control-programs
func createAccountControlProgram(ctx context.Context, accountID string) (interface{}, error) {
	controlProgram, err := accounts.CreateControlProgram(ctx, accountID)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"control_program": controlProgram,
	}
	return ret, nil
}
