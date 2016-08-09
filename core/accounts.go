package core

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/account"
	"chain/metrics"
)

// POST /list-accounts
func listAccounts(ctx context.Context, in requestQuery) (result page, err error) {
	limit := defAccountPageSize

	accounts, cursor, err := account.List(ctx, in.Cursor, limit)
	if err != nil {
		return result, err
	}

	for _, account := range accounts {
		result.Items = append(result.Items, account)
	}
	result.LastPage = len(accounts) < limit
	result.Query.Cursor = cursor
	return result, nil
}

// POST /create-account
// TODO(boymanjor): Refactor for batch creation
func createAccount(ctx context.Context, in struct {
	XPubs  []string
	Quorum int
	Tags   map[string]interface{}

	// ClientToken is the application's unique token for the account. Every account
	// should have a unique client token. The client token is used to ensure
	// idempotency of create account requests. Duplicate create account requests
	// with the same client_token will only create one account.
	ClientToken *string `json:"client_token"`
}) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())

	return account.Create(ctx, in.XPubs, in.Quorum, in.Tags, in.ClientToken)
}

// POST /get-account
// TODO(boymanjor): Refactor for batch retrieval
func getAccount(ctx context.Context, in struct{ ID string }) (interface{}, error) {
	return account.Find(ctx, in.ID)
}

// POST /set-account-tags
func setAccountTags(ctx context.Context, in struct {
	AccountID string `json:"account_id"`
	Tags      map[string]interface{}
}) (interface{}, error) {
	return account.SetTags(ctx, in.AccountID, in.Tags)
}

// DELETE /v3/accounts/:accountID
// Idempotent
func archiveAccount(ctx context.Context, accountID string) error {
	return account.Archive(ctx, accountID)
}
