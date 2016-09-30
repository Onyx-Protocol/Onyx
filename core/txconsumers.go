package core

import (
	"context"
	"fmt"
	"math"

	"chain/core/query"
	"chain/core/txconsumer"
	"chain/errors"
	"chain/net/http/httpjson"
)

// POST /create-txconsumer
func (a *api) createTxConsumer(ctx context.Context, in struct {
	Alias  string
	Filter string

	// ClientToken is the application's unique token for the txconsumer. Every txconsumer
	// should have a unique client token. The client token is used to ensure
	// idempotency of create txconsumer requests. Duplicate create txconsumer requests
	// with the same client_token will only create one txconsumer.
	ClientToken *string `json:"client_token"`
}) (*txconsumer.TxConsumer, error) {
	after := fmt.Sprintf("%d:%d-%d", a.c.Height(), math.MaxInt32, uint64(math.MaxInt64))
	return txconsumer.Create(ctx, in.Alias, in.Filter, after, in.ClientToken)
}

// POST /get-transaction-consumer
func getTxConsumer(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}) (*txconsumer.TxConsumer, error) {
	return txconsumer.Find(ctx, in.ID, in.Alias)
}

// POST /delete-transaction-consumer
func deleteTxConsumer(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}) error {
	return txconsumer.Delete(ctx, in.ID, in.Alias)
}

// POST /update-transaction-consumer
func updateTxConsumer(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
	Prev  string `json:"previous_after"`
	After string `json:"after"`
}) (*txconsumer.TxConsumer, error) {
	// TODO(tessr): Consider moving this function into the txconsumer package.
	// (It's currently outside the txconsumer package to avoid a dependecy cycle
	// between txconsumer and query.)
	bad, err := txAfterIsBefore(in.After, in.Prev)
	if err != nil {
		return nil, err
	}

	if bad {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "new After cannot be before Prev")
	}

	return txconsumer.Update(ctx, in.ID, in.Alias, in.Prev, in.After)
}

// txAfterIsBefore returns true if a is before b. It returns an error if either
// a or b are not valid query.TxAfters.
func txAfterIsBefore(a, b string) (bool, error) {
	aAfter, err := query.DecodeTxAfter(a)
	if err != nil {
		return false, err
	}

	bAfter, err := query.DecodeTxAfter(b)
	if err != nil {
		return false, err
	}

	return aAfter.FromBlockHeight < bAfter.FromBlockHeight ||
		(aAfter.FromBlockHeight == bAfter.FromBlockHeight &&
			aAfter.FromPosition < bAfter.FromPosition), nil
}
