package core

import (
	"context"
	"math"

	"chain/core/query"
	"chain/core/query/filter"
	"chain/errors"
	"chain/net/http/httpjson"
)

// listAccounts is an http handler for listing accounts matching
// an index or an ad-hoc filter.
//
// POST /list-accounts
func (a *API) listAccounts(ctx context.Context, in requestQuery) (page, error) {
	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}
	after := in.After

	// Use the filter engine for querying account tags.
	accounts, after, err := a.indexer.Accounts(ctx, in.Filter, in.FilterParams, after, limit)
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

// listAssets is an http handler for listing assets matching
// an index or an ad-hoc filter.
//
// POST /list-assets
func (a *API) listAssets(ctx context.Context, in requestQuery) (page, error) {
	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}
	after := in.After

	// Use the query engine for querying asset tags.
	assets, after, err := a.indexer.Assets(ctx, in.Filter, in.FilterParams, after, limit)
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

// POST /list-balances
func (a *API) listBalances(ctx context.Context, in requestQuery) (result page, err error) {
	var sumBy []filter.Field

	// Since an empty SumBy yields a meaningless result, we'll provide a
	// sensible default here.
	if len(in.SumBy) == 0 {
		in.SumBy = []string{"asset_alias", "asset_id"}
	}

	for _, field := range in.SumBy {
		f, err := filter.ParseField(field)
		if err != nil {
			return result, err
		}
		sumBy = append(sumBy, f)
	}

	timestampMS := in.TimestampMS
	if timestampMS == 0 {
		timestampMS = math.MaxInt64
	} else if timestampMS > math.MaxInt64 {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "timestamp is too large")
	}

	// TODO(jackson): paginate this endpoint.
	balances, err := a.indexer.Balances(ctx, in.Filter, in.FilterParams, sumBy, timestampMS)
	if err != nil {
		return result, err
	}

	result.Items = httpjson.Array(balances)
	result.LastPage = true
	result.Next = in
	return result, nil
}

// listTransactions is an http handler for listing transactions matching
// an index or an ad-hoc filter.
//
// POST /list-transactions
func (a *API) listTransactions(ctx context.Context, in requestQuery) (result page, err error) {
	var c context.CancelFunc
	timeout := in.Timeout.Duration
	if timeout != 0 {
		ctx, c = context.WithTimeout(ctx, timeout)
		defer c()
	}

	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	endTimeMS := in.EndTimeMS
	if endTimeMS == 0 {
		endTimeMS = math.MaxInt64
	} else if endTimeMS > math.MaxInt64 {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "end timestamp is too large")
	}

	// Either parse the provided `after` or look one up for the time range.
	var after query.TxAfter
	if in.After != "" {
		after, err = query.DecodeTxAfter(in.After)
		if err != nil {
			return result, errors.Wrap(err, "decoding `after`")
		}
	} else {
		after, err = a.indexer.LookupTxAfter(ctx, in.StartTimeMS, endTimeMS)
		if err != nil {
			return result, err
		}
	}

	txns, nextAfter, err := a.indexer.Transactions(ctx, in.Filter, in.FilterParams, after, limit, in.AscLongPoll)
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

// listTxFeeds is an http handler for listing txfeeds. It does not take a filter.
//
// POST /list-transaction-feeds
func (a *API) listTxFeeds(ctx context.Context, in requestQuery) (page, error) {
	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	after := in.After

	txfeeds, after, err := a.txFeeds.Query(ctx, after, limit)
	if err != nil {
		return page{}, errors.Wrap(err, "running txfeed query")
	}

	out := in
	out.After = after
	return page{
		Items:    httpjson.Array(txfeeds),
		LastPage: len(txfeeds) < limit,
		Next:     out,
	}, nil
}

// POST /list-unspent-outputs
func (a *API) listUnspentOutputs(ctx context.Context, in requestQuery) (result page, err error) {
	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	var after *query.OutputsAfter
	if in.After != "" {
		after, err = query.DecodeOutputsAfter(in.After)
		if err != nil {
			return result, errors.Wrap(err, "decoding `after`")
		}
	}

	timestampMS := in.TimestampMS
	if timestampMS == 0 {
		timestampMS = math.MaxInt64
	} else if timestampMS > math.MaxInt64 {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "timestamp is too large")
	}
	outputs, nextAfter, err := a.indexer.Outputs(ctx, in.Filter, in.FilterParams, timestampMS, after, limit)
	if err != nil {
		return result, errors.Wrap(err, "querying outputs")
	}

	outQuery := in
	outQuery.After = nextAfter.String()
	return page{
		Items:    httpjson.Array(outputs),
		LastPage: len(outputs) < limit,
		Next:     outQuery,
	}, nil
}
