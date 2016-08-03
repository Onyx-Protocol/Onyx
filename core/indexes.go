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
	if in.Type != "transaction" && in.Type != "output" {
		return nil, errors.WithDetailf(httpjson.ErrBadRequest, "unknown index type %q", in.Type)
	}

	idx, err := a.indexer.CreateIndex(ctx, in.ID, in.Type, in.Query)
	return idx, errors.Wrap(err, "creating the new index")
}

// listIndexes is an http handler for listing CQL indexes.
//
// POST /list-indexes
func (a *api) listIndexes(ctx context.Context) ([]*query.Index, error) {
	indexes, err := a.indexer.ListIndexes(ctx)
	return indexes, errors.Wrap(err, "listing indexes")
}
