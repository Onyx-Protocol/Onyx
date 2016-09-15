package core

import (
	"context"
	"time"

	"chain/core/query"
	"chain/core/query/filter"
	"chain/errors"
	"chain/net/http/httpjson"
	"chain/protocol/bc"
)

// listTransactions is an http handler for listing transactions matching
// an index or an ad-hoc filter.
//
// POST /list-transactions
func (a *api) listTransactions(ctx context.Context, in requestQuery) (result page, err error) {
	if in.EndTimeMS == 0 {
		in.EndTimeMS = bc.Millis(time.Now())
	}

	var (
		p     filter.Predicate
		after query.TxAfter
	)

	// Build the filter predicate.
	p, err = filter.Parse(in.Filter)
	if err != nil {
		return result, err
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
	if in.TimestampMS == 0 {
		in.TimestampMS = bc.Millis(time.Now())
	}

	var p filter.Predicate
	var sumBy []filter.Field
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
	p, err = filter.Parse(in.Filter)
	if err != nil {
		return result, err
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
