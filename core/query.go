package core

import (
	"fmt"

	"golang.org/x/net/context"

	"chain/core/account"
	"chain/core/query"
	"chain/core/query/chql"
	"chain/errors"
	"chain/net/http/httpjson"
)

var (
	ErrBadIndexConfig = errors.New("index configuration invalid")
)

// createIndex is an http handler for creating indexes.
//
// POST /create-index
func (a *api) createIndex(ctx context.Context, in struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Query    string `json:"query"`
	Unspents bool   `json:"unspents"`
}) (*query.Index, error) {
	if in.Type != "transaction" && in.Type != "balance" && in.Type != "asset" {
		return nil, errors.WithDetailf(ErrBadIndexConfig, "unknown index type %q", in.Type)
	}
	if in.Unspents && in.Type != "balance" {
		return nil, errors.WithDetail(ErrBadIndexConfig, "unspents flag is only valid for balance indexes")
	}

	idx, err := a.indexer.CreateIndex(ctx, in.ID, in.Type, in.Query, in.Unspents)
	return idx, errors.Wrap(err, "creating the new index")
}

// listIndexes is an http handler for listing ChQL indexes.
//
// POST /list-indexes
func (a *api) listIndexes(ctx context.Context, query requestQuery) (page, error) {
	limit := defGenericPageSize

	indexes, cursor, err := a.indexer.ListIndexes(ctx, query.Cursor, limit)
	if err != nil {
		return page{}, errors.Wrap(err, "listing indexes")
	}

	query.Cursor = cursor
	return page{
		Items:    httpjson.Array(indexes),
		LastPage: len(indexes) < limit,
		Query:    query,
	}, nil
}

// listTransactions is an http handler for listing transactions matching
// a ChQL query or index.
//
// POST /list-transactions
func (a *api) listTransactions(ctx context.Context, in requestQuery) (result page, err error) {
	if in.Index != "" && in.ChQL != "" {
		return result, fmt.Errorf("cannot provide both index and query")
	}

	var (
		q   chql.Query
		cur *query.TxCursor
	)

	// Build the ChQL query
	if in.Index != "" {
		idx, err := a.indexer.GetIndex(ctx, in.Index, "transaction")
		if err != nil {
			return result, err
		}
		if idx == nil {
			return result, fmt.Errorf("Unknown transaction index %q", in.Index)
		}
		q = idx.Query
	} else {
		q, err = chql.Parse(in.ChQL)
		if err != nil {
			return result, err
		}
	}

	// Either parse the provided cursor or look one up for the time range.
	if in.Cursor != "" {
		cur, err = query.DecodeTxCursor(in.Cursor)
		if err != nil {
			return result, errors.Wrap(err, "decoding cursor")
		}
	}
	if cur == nil {
		cur, err = a.indexer.LookupTxCursor(ctx, in.StartTime, in.EndTime)
		if err != nil {
			return result, err
		}
	}
	if cur == nil { // no results; empty page
		return page{LastPage: true}, nil
	}

	limit := defGenericPageSize
	txns, nextCur, err := a.indexer.Transactions(ctx, q, in.ChQLParams, *cur, limit)
	if err != nil {
		return result, errors.Wrap(err, "running tx query")
	}

	out := in
	out.Cursor = nextCur.String()
	return page{
		Items:    httpjson.Array(txns),
		LastPage: len(txns) < limit,
		Query:    out,
	}, nil
}

// listAccounts is an http handler for listing accounts matching
// a ChQL query or index.
//
// TODO(jackson): This endpoint performs two separate db queries, one
// for performing ChQL query filtering, the other for retrieving
// account/signer data. We might want to refactor this, but for now
// it maintains a nice boundary between the core/query and core/account
// packages.
//
// POST /list-accounts
func (a *api) listAccounts(ctx context.Context, in requestQuery) (result page, err error) {
	limit := defGenericPageSize

	// Build the ChQL query
	q, err := chql.Parse(in.ChQL)
	if err != nil {
		return result, err
	}
	cur := in.Cursor

	// Use the ChQL query engine for querying account tags.
	var accountIDs []string
	var accountTags map[string]map[string]interface{}
	accountIDs, accountTags, cur, err = a.indexer.AccountTags(ctx, q, in.ChQLParams, cur, limit)
	if err != nil {
		return result, errors.Wrap(err, "running acc query")
	}

	// Pull in the accounts by the IDs.
	accounts, err := account.FindBatch(ctx, accountIDs...)
	if err != nil {
		return result, errors.Wrap(err, "retrieving account list")
	}
	items := make([]*account.Account, 0, len(accountIDs))
	for _, id := range accountIDs {
		account := accounts[id]
		account.Tags = accountTags[id]
		items = append(items, account)
	}

	out := in
	out.Cursor = cur
	return page{
		Items:    httpjson.Array(items),
		LastPage: len(accountIDs) < limit,
		Query:    out,
	}, nil
}
