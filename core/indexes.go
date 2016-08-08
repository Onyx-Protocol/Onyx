package core

import (
	"golang.org/x/net/context"

	"chain/core/query"
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
func (a *api) listIndexes(ctx context.Context, in requestQuery) (result page, err error) {
	limit := defGenericPageSize

	indexes, cursor, err := a.indexer.ListIndexes(ctx, in.Cursor, limit)
	if err != nil {
		return result, errors.Wrap(err, "listing indexes")
	}
	for _, item := range indexes {
		result.Items = append(result.Items, item)
	}
	result.LastPage = len(indexes) < limit
	result.Query.Cursor = cursor
	return result, nil
}

// getIndex is an http handler for retrieving a ChQL index.
//
// POST /get-index
func (a *api) getIndex(ctx context.Context, in struct{ ID string }) (*query.Index, error) {
	idx, err := a.indexer.GetIndex(ctx, in.ID)
	return idx, errors.Wrap(err, "retrieving an index")
}
