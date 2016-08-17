package core

import (
	"context"
	"time"

	"chain/core/query"
	"chain/core/query/chql"
	"chain/cos/bc"
	"chain/database/pg"
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
	Alias    string `json:"alias"`
	Type     string `json:"type"`
	Query    string `json:"query"`
	Unspents bool   `json:"unspents"`
}) (*query.Index, error) {
	if !query.IndexTypes[in.Type] {
		return nil, errors.WithDetailf(ErrBadIndexConfig, "unknown index type %q", in.Type)
	}
	if in.Unspents && in.Type != query.IndexTypeBalance {
		return nil, errors.WithDetail(ErrBadIndexConfig, "unspents flag is only valid for balance indexes")
	}
	if in.Alias == "" {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "missing index alias")
	}

	idx, err := a.indexer.CreateIndex(ctx, in.Alias, in.Type, in.Query, in.Unspents)
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

var (
	ErrNeitherIndexNorQuery = errors.New("must provide either index or query")
	ErrBothIndexAndQuery    = errors.New("cannot provide both index and query")
)

// listTransactions is an http handler for listing transactions matching
// a ChQL query or index.
//
// POST /list-transactions
func (a *api) listTransactions(ctx context.Context, in requestQuery) (result page, err error) {
	if (in.IndexID != "" || in.IndexAlias != "") && in.ChQL != "" {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "cannot provide both index and query")
	}
	if in.EndTimeMS == 0 {
		in.EndTimeMS = bc.Millis(time.Now())
	}

	var (
		q   chql.Query
		cur query.TxCursor
	)

	// Build the ChQL query
	if in.IndexAlias != "" || in.IndexID != "" {
		idx, err := a.indexer.GetIndex(ctx, in.IndexID, in.IndexAlias, query.IndexTypeTransaction)
		if err != nil {
			return result, err
		}
		if idx == nil {
			return result, errors.WithDetail(pg.ErrUserInputNotFound, "transaction index not found")
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
	} else {
		cur, err = a.indexer.LookupTxCursor(ctx, in.StartTimeMS, in.EndTimeMS)
		if err != nil {
			return result, err
		}
	}

	limit := defGenericPageSize
	txns, nextCur, err := a.indexer.Transactions(ctx, q, in.ChQLParams, cur, limit)
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
// POST /list-accounts
func (a *api) listAccounts(ctx context.Context, in requestQuery) (page, error) {
	limit := defGenericPageSize

	// Build the ChQL query
	q, err := chql.Parse(in.ChQL)
	if err != nil {
		return page{}, errors.Wrap(err, "parsing acc query")
	}
	cur := in.Cursor

	// Use the ChQL query engine for querying account tags.
	accounts, cur, err := a.indexer.Accounts(ctx, q, in.ChQLParams, cur, limit)
	if err != nil {
		return page{}, errors.Wrap(err, "running acc query")
	}

	// Pull in the accounts by the IDs
	out := in
	out.Cursor = cur
	return page{
		Items:    httpjson.Array(accounts),
		LastPage: len(accounts) < limit,
		Query:    out,
	}, nil
}

// POST /list-balances
func (a *api) listBalances(ctx context.Context, in requestQuery) (result page, err error) {
	if (in.IndexID != "" || in.IndexAlias != "") && in.ChQL != "" {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "cannot provide both index and query")
	}
	if in.TimestampMS == 0 {
		in.TimestampMS = bc.Millis(time.Now())
	}

	var q chql.Query
	if in.IndexID != "" || in.IndexAlias != "" {
		idx, err := a.indexer.GetIndex(ctx, in.IndexID, in.IndexAlias, query.IndexTypeBalance)
		if err != nil {
			return result, err
		}
		if idx == nil {
			return result, errors.WithDetail(pg.ErrUserInputNotFound, "balance index not found")
		}
		q = idx.Query
	} else {
		q, err = chql.Parse(in.ChQL)
		if err != nil {
			return result, err
		}
	}

	// TODO(jackson): paginate this endpoint.
	balances, err := a.indexer.Balances(ctx, q, in.ChQLParams, in.TimestampMS)
	if err != nil {
		return result, err
	}

	result.Items = httpjson.Array(balances)
	result.LastPage = true
	result.Query = in
	return result, nil
}

// POST /list-unspent-outputs
func (a *api) listUnspentOutputs(ctx context.Context, in requestQuery) (result page, err error) {
	if (in.IndexID != "" || in.IndexAlias != "") && in.ChQL != "" {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "cannot provide both index and query")
	}

	var q chql.Query
	if in.IndexID != "" || in.IndexAlias != "" {
		idx, err := a.indexer.GetIndex(ctx, in.IndexID, in.IndexAlias, query.IndexTypeBalance)
		if err != nil {
			return result, err
		}
		if idx == nil {
			return result, errors.WithDetail(pg.ErrUserInputNotFound, "balance index not found")
		}
		if !idx.Unspents {
			return result, errors.WithDetail(httpjson.ErrBadRequest, "balance index doesn't support output indexing")
		}
		q = idx.Query
	} else {
		q, err = chql.Parse(in.ChQL)
		if err != nil {
			return result, err
		}
	}

	var cursor *query.OutputsCursor
	if in.Cursor != "" {
		cursor, err = query.DecodeOutputsCursor(in.Cursor)
		if err != nil {
			return result, errors.Wrap(err, "decoding cursor")
		}
	}

	limit := defGenericPageSize
	outputs, newCursor, err := a.indexer.Outputs(ctx, q, in.ChQLParams, in.TimestampMS, cursor, limit)

	outQuery := in
	outQuery.Cursor = newCursor.String()
	return page{
		Items:    outputs,
		LastPage: len(outputs) < limit,
		Query:    outQuery,
	}, nil
}

// listAssets is an http handler for listing assets matching
// a ChQL query or index.
//
// POST /list-assets
func (a *api) listAssets(ctx context.Context, in requestQuery) (page, error) {
	limit := defGenericPageSize

	// Build the ChQL query
	q, err := chql.Parse(in.ChQL)
	if err != nil {
		return page{}, err
	}
	cur := in.Cursor

	// Use the ChQL query engine for querying asset tags.
	var assets []map[string]interface{}
	assets, cur, err = a.indexer.Assets(ctx, q, in.ChQLParams, cur, limit)
	if err != nil {
		return page{}, errors.Wrap(err, "running asset query")
	}

	out := in
	out.Cursor = cur
	return page{
		Items:    httpjson.Array(assets),
		LastPage: len(assets) < limit,
		Query:    out,
	}, nil
}
