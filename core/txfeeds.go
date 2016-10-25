package core

import (
	"context"
	"fmt"
	"math"

	"chain/core/query"
	"chain/core/txfeed"
	"chain/errors"
	"chain/net/http/httpjson"
)

// POST /create-txfeed
func (h *Handler) createTxFeed(ctx context.Context, in struct {
	Alias  string
	Filter string

	// ClientToken is the application's unique token for the txfeed. Every txfeed
	// should have a unique client token. The client token is used to ensure
	// idempotency of create txfeed requests. Duplicate create txfeed requests
	// with the same client_token will only create one txfeed.
	ClientToken *string `json:"client_token"`
}) (*txfeed.TxFeed, error) {
	after := fmt.Sprintf("%d:%d-%d", h.Chain.Height(), math.MaxInt32, uint64(math.MaxInt64))
	return h.TxFeeds.Create(ctx, in.Alias, in.Filter, after, in.ClientToken)
}

// POST /get-transaction-feed
func (h *Handler) getTxFeed(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}) (*txfeed.TxFeed, error) {
	return h.TxFeeds.Find(ctx, in.ID, in.Alias)
}

// POST /delete-transaction-feed
func (h *Handler) deleteTxFeed(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
}) error {
	return h.TxFeeds.Delete(ctx, in.ID, in.Alias)
}

// POST /update-transaction-feed
func (h *Handler) updateTxFeed(ctx context.Context, in struct {
	ID    string `json:"id,omitempty"`
	Alias string `json:"alias,omitempty"`
	Prev  string `json:"previous_after"`
	After string `json:"after"`
}) (*txfeed.TxFeed, error) {
	// TODO(tessr): Consider moving this function into the txfeed package.
	// (It's currently outside the txfeed package to avoid a dependecy cycle
	// between txfeed and query.)
	bad, err := txAfterIsBefore(in.After, in.Prev)
	if err != nil {
		return nil, err
	}

	if bad {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "new After cannot be before Prev")
	}

	return h.TxFeeds.Update(ctx, in.ID, in.Alias, in.After, in.Prev)
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
