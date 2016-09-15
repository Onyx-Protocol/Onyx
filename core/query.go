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
	errBadIndexConfig = errors.New("index configuration invalid")
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
		return nil, errors.WithDetailf(errBadIndexConfig, "unknown index type %q", in.Type)
	}
	if len(in.SumBy) > 0 && in.Type != query.IndexTypeBalance {
		return nil, errors.WithDetail(errBadIndexConfig, "sum-by field is only valid for balance indexes")
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

	indexes, after, err := a.indexer.ListIndexes(ctx, query.After, limit)
	if err != nil {
		return page{}, errors.Wrap(err, "listing indexes")
	}

	query.After = after
	return page{
		Items:    httpjson.Array(indexes),
		LastPage: len(indexes) < limit,
		Next:     query,
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
		p     filter.Predicate
		after query.TxAfter
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

	// Either parse the provided `after` or look one up for the time range.
	if in.After != "" {
		after, err = query.DecodeTxAfter(in.After)
		if err != nil {
			return result, errors.Wrap(err, "decoding `after`")
		}
	} else {
		after, err = a.indexer.LookupTxAfter(ctx, in.StartTimeMS, in.EndTimeMS)
		if err != nil {
			return result, err
		}
	}

	var asc bool
	if in.Order == "asc" {
		asc = true
	}

	limit := defGenericPageSize
	txns, nextAfter, err := a.indexer.Transactions(ctx, p, in.FilterParams, after, limit, asc)
	if err != nil {
		return result, errors.Wrap(err, "running tx query")
	}

	out := in
	out.After = nextAfter.String()
	return page{
		Items:    httpjson.Array(txns),
		LastPage: len(txns) < limit,
		Next:     out,
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
	after := in.After

	// Use the filter engine for querying account tags.
	accounts, after, err := a.indexer.Accounts(ctx, p, in.FilterParams, after, limit)
	if err != nil {
		return page{}, errors.Wrap(err, "running acc query")
	}

	// Pull in the accounts by the IDs
	out := in
	out.After = after
	return page{
		Items:    httpjson.Array(accounts),
		LastPage: len(accounts) < limit,
		Next:     out,
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
	result.Next = in
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

	var after *query.OutputsAfter
	if in.After != "" {
		after, err = query.DecodeOutputsAfter(in.After)
		if err != nil {
			return result, errors.Wrap(err, "decoding `after`")
		}
	}

	limit := defGenericPageSize
	outputs, nextAfter, err := a.indexer.Outputs(ctx, p, in.FilterParams, in.TimestampMS, after, limit)
	if err != nil {
		return result, errors.Wrap(err, "querying outputs")
	}

	outQuery := in
	outQuery.After = nextAfter.String()
	return page{
		Items:    outputs,
		LastPage: len(outputs) < limit,
		Next:     outQuery,
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
	after := in.After

	// Use the query engine for querying asset tags.
	var assets []map[string]interface{}
	assets, after, err = a.indexer.Assets(ctx, p, in.FilterParams, after, limit)
	if err != nil {
		return page{}, errors.Wrap(err, "running asset query")
	}

	out := in
	out.After = after
	return page{
		Items:    httpjson.Array(assets),
		LastPage: len(assets) < limit,
		Next:     out,
	}, nil
}
