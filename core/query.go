package core

import (
	"context"
	"time"

	"chain/core/query"
	"chain/core/query/filter"
	"chain/database/pg"
	"chain/errors"
	"chain/net/http/httpjson"
	"chain/protocol/bc"
)

var (
	ErrBadIndexConfig = errors.New("index configuration invalid")
)

// createIndex is an http handler for creating indexes.
//
// POST /create-index
func (a *api) createIndex(ctx context.Context, in struct {
	Alias  string   `json:"alias"`
	Type   string   `json:"type"`
	Filter string   `json:"filter"`
	SumBy  []string `json:"sum_by"`
}) (*query.Index, error) {
	if !query.IndexTypes[in.Type] {
		return nil, errors.WithDetailf(ErrBadIndexConfig, "unknown index type %q", in.Type)
	}
	if len(in.SumBy) > 0 && in.Type != query.IndexTypeBalance {
		return nil, errors.WithDetail(ErrBadIndexConfig, "sum-by field is only valid for balance indexes")
	}
	if in.Alias == "" {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "missing index alias")
	}

	idx, err := a.indexer.CreateIndex(ctx, in.Alias, in.Type, in.Filter, in.SumBy)
	return idx, errors.Wrap(err, "creating the new index")
}

// listIndexes is an http handler for listing search indexes.
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
// an index or an ad-hoc filter.
//
// POST /list-transactions
func (a *api) listTransactions(ctx context.Context, in requestQuery) (result page, err error) {
	if (in.IndexID != "" || in.IndexAlias != "") && in.Filter != "" {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "cannot provide both index and filter predicate")
	}
	if in.EndTimeMS == 0 {
		in.EndTimeMS = bc.Millis(time.Now())
	}

	var (
		p   filter.Predicate
		cur query.TxCursor
	)

	// Build the filter predicate.
	if in.IndexAlias != "" || in.IndexID != "" {
		idx, err := a.indexer.GetIndex(ctx, in.IndexID, in.IndexAlias, query.IndexTypeTransaction)
		if err != nil {
			return result, err
		}
		if idx == nil {
			return result, errors.WithDetail(pg.ErrUserInputNotFound, "transaction index not found")
		}
		p = idx.Predicate
	} else {
		p, err = filter.Parse(in.Filter)
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
	txns, nextCur, err := a.indexer.Transactions(ctx, p, in.FilterParams, cur, limit)
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
// an index or an ad-hoc filter.
//
// POST /list-accounts
func (a *api) listAccounts(ctx context.Context, in requestQuery) (page, error) {
	limit := defGenericPageSize

	// Build the filter predicate.
	p, err := filter.Parse(in.Filter)
	if err != nil {
		return page{}, errors.Wrap(err, "parsing acc query")
	}
	cur := in.Cursor

	// Use the filter engine for querying account tags.
	accounts, cur, err := a.indexer.Accounts(ctx, p, in.FilterParams, cur, limit)
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
	if (in.IndexID != "" || in.IndexAlias != "") && in.Filter != "" {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "cannot provide both index and filter predicate")
	}
	if in.TimestampMS == 0 {
		in.TimestampMS = bc.Millis(time.Now())
	}

	var p filter.Predicate
	var sumBy []filter.Field
	if in.IndexID != "" || in.IndexAlias != "" {
		idx, err := a.indexer.GetIndex(ctx, in.IndexID, in.IndexAlias, query.IndexTypeBalance)
		if err != nil {
			return result, err
		}
		if idx == nil {
			return result, errors.WithDetail(pg.ErrUserInputNotFound, "balance index not found")
		}
		p = idx.Predicate
		sumBy = idx.SumBy
	} else {
		p, err = filter.Parse(in.Filter)
		if err != nil {
			return result, err
		}

		for _, field := range in.SumBy {
			f, err := filter.ParseField(field)
			if err != nil {
				return result, err
			}
			sumBy = append(sumBy, f)
		}
	}

	// TODO(jackson): paginate this endpoint.
	balances, err := a.indexer.Balances(ctx, p, in.FilterParams, sumBy, in.TimestampMS)
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
	if in.TimestampMS == 0 {
		in.TimestampMS = bc.Millis(time.Now())
	}
	var p filter.Predicate
	if in.IndexID != "" || in.IndexAlias != "" {
		idx, err := a.indexer.GetIndex(ctx, in.IndexID, in.IndexAlias, query.IndexTypeOutput)
		if err != nil {
			return result, err
		}
		if idx == nil {
			return result, errors.WithDetail(pg.ErrUserInputNotFound, "output index not found")
		}
		p = idx.Predicate
	} else {
		p, err = filter.Parse(in.Filter)
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
	outputs, newCursor, err := a.indexer.Outputs(ctx, p, in.FilterParams, in.TimestampMS, cursor, limit)
	if err != nil {
		return result, errors.Wrap(err, "querying outputs")
	}

	outQuery := in
	outQuery.Cursor = newCursor.String()
	return page{
		Items:    outputs,
		LastPage: len(outputs) < limit,
		Query:    outQuery,
	}, nil
}

// listAssets is an http handler for listing assets matching
// an index or an ad-hoc filter.
//
// POST /list-assets
func (a *api) listAssets(ctx context.Context, in requestQuery) (page, error) {
	limit := defGenericPageSize

	// Build the filter predicate.
	p, err := filter.Parse(in.Filter)
	if err != nil {
		return page{}, err
	}
	cur := in.Cursor

	// Use the query engine for querying asset tags.
	var assets []map[string]interface{}
	assets, cur, err = a.indexer.Assets(ctx, p, in.FilterParams, cur, limit)
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
