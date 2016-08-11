package core

import (
	"fmt"

	"golang.org/x/net/context"

	"chain/core/query"
	"chain/core/query/chql"
	"chain/errors"
	"chain/net/http/httpjson"
)

// createIndex is an http handler for creating indexes.
//
// POST /create-index
func (a *api) createIndex(ctx context.Context, in struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Query string `json:"query"`
}) (*query.Index, error) {
	if in.Type != "transaction" && in.Type != "balance" && in.Type != "asset" {
		return nil, errors.WithDetailf(httpjson.ErrBadRequest, "unknown index type %q", in.Type)
	}

	idx, err := a.indexer.CreateIndex(ctx, in.ID, in.Type, in.Query)
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
	if in.Index == "" && in.Query == "" {
		return result, fmt.Errorf("must provide either index or query")
	} else if in.Index != "" && in.Query != "" {
		return result, fmt.Errorf("cannot provide both index and query")
	}

	var (
		q   chql.Query
		cur *query.TxCursor
	)

	// Build the ChQL query
	if in.Query != "" {
		q, err = chql.Parse(in.Query)
		if err != nil {
			return result, err
		}
	} else {
		idx, err := a.indexer.GetIndex(ctx, in.Index, "transaction")
		if err != nil {
			return result, err
		}
		if idx == nil {
			return result, fmt.Errorf("Unknown transaction index %q", in.Index)
		}
		q = idx.Query
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
	txns, nextCur, err := a.indexer.Transactions(ctx, q, in.Params, *cur, limit)
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
