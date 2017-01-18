package core

import (
	"context"
	"encoding/json"
	"math"

	"chain/core/query"
	"chain/core/query/filter"
	"chain/errors"
	"chain/net/http/httpjson"
)

// These types enforce the ordering of JSON fields in API output.
type (
	txinResp struct {
		Type            string           `json:"type"`
		AssetID         string           `json:"asset_id"`
		AssetAlias      string           `json:"asset_alias,omitempty"`
		AssetDefinition *json.RawMessage `json:"asset_definition"`
		AssetTags       *json.RawMessage `json:"asset_tags,omitempty"`
		AssetIsLocal    string           `json:"asset_is_local"`
		Amount          uint64           `json:"amount"`
		IssuanceProgram string           `json:"issuance_program,omitempty"`
		SpentOutput     *struct {
			TransactionID string `json:"transaction_id"`
			Position      uint32 `json:"position"`
		} `json:"spent_output,omitempty"`
		*txAccount
		ReferenceData *json.RawMessage `json:"reference_data"`
		IsLocal       string           `json:"is_local"`
	}
	txoutResp struct {
		Type            string           `json:"type"`
		Purpose         string           `json:"purpose,omitempty"`
		Position        uint32           `json:"position"`
		AssetID         string           `json:"asset_id"`
		AssetAlias      string           `json:"asset_alias,omitempty"`
		AssetDefinition *json.RawMessage `json:"asset_definition"`
		AssetTags       *json.RawMessage `json:"asset_tags"`
		AssetIsLocal    string           `json:"asset_is_local"`
		Amount          uint64           `json:"amount"`
		*txAccount
		ControlProgram string           `json:"control_program"`
		ReferenceData  *json.RawMessage `json:"reference_data"`
		IsLocal        string           `json:"is_local"`
	}
	txResp struct {
		ID            string           `json:"id"`
		Timestamp     string           `json:"timestamp"`
		BlockID       string           `json:"block_id"`
		BlockHeight   uint64           `json:"block_height"`
		Position      uint32           `json:"position"`
		ReferenceData *json.RawMessage `json:"reference_data"`
		IsLocal       string           `json:"is_local"`
		Inputs        []*txinResp      `json:"inputs"`
		Outputs       []*txoutResp     `json:"outputs"`
	}
	txAccount struct {
		AccountID    string           `json:"account_id"`
		AccountAlias string           `json:"account_alias,omitempty"`
		AccountTags  *json.RawMessage `json:"account_tags"`
	}
)

// listTransactions is an http handler for listing transactions matching
// an index or an ad-hoc filter.
//
// POST /list-transactions
func (h *Handler) listTransactions(ctx context.Context, in requestQuery) (result page, err error) {
	var c context.CancelFunc
	timeout := in.Timeout.Duration
	if timeout != 0 {
		ctx, c = context.WithTimeout(ctx, timeout)
		defer c()
	}
	var (
		p     filter.Predicate
		after query.TxAfter
	)

	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	// Build the filter predicate.
	p, err = filter.Parse(in.Filter)
	if err != nil {
		return result, err
	}

	endTimeMS := in.EndTimeMS
	if endTimeMS == 0 {
		endTimeMS = math.MaxInt64
	} else if endTimeMS > math.MaxInt64 {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "end timestamp is too large")
	}
	// Either parse the provided `after` or look one up for the time range.
	if in.After != "" {
		after, err = query.DecodeTxAfter(in.After)
		if err != nil {
			return result, errors.Wrap(err, "decoding `after`")
		}
	} else {
		after, err = h.Indexer.LookupTxAfter(ctx, in.StartTimeMS, endTimeMS)
		if err != nil {
			return result, err
		}
	}

	txns, nextAfter, err := h.Indexer.Transactions(ctx, p, in.FilterParams, after, limit, in.AscLongPoll)
	if err != nil {
		return result, errors.Wrap(err, "running tx query")
	}

	resp := make([]*txResp, 0, len(txns))
	for _, t := range txns {
		var r txResp
		err := json.Unmarshal(t, &r)
		if err != nil {
			return result, errors.Wrap(err, "unmarshaling stored transaction")
		}
		resp = append(resp, &r)
	}

	out := in
	out.After = nextAfter.String()
	return page{
		Items:    httpjson.Array(resp),
		LastPage: len(resp) < limit,
		Next:     out,
	}, nil
}

// listAccounts is an http handler for listing accounts matching
// an index or an ad-hoc filter.
//
// POST /list-accounts
func (h *Handler) listAccounts(ctx context.Context, in requestQuery) (page, error) {
	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	// Build the filter predicate.
	p, err := filter.Parse(in.Filter)
	if err != nil {
		return page{}, errors.Wrap(err, "parsing acc query")
	}
	after := in.After

	// Use the filter engine for querying account tags.
	accounts, after, err := h.Indexer.Accounts(ctx, p, in.FilterParams, after, limit)
	if err != nil {
		return page{}, errors.Wrap(err, "running acc query")
	}

	result := make([]*accountResponse, 0, len(accounts))
	for _, a := range accounts {
		var r accountResponse
		err := json.Unmarshal(a, &r)
		if err != nil {
			return page{}, errors.Wrap(err, "unmarshaling stored account")
		}
		result = append(result, &r)
	}

	// Pull in the accounts by the IDs
	out := in
	out.After = after
	return page{
		Items:    httpjson.Array(result),
		LastPage: len(result) < limit,
		Next:     out,
	}, nil
}

// POST /list-balances
func (h *Handler) listBalances(ctx context.Context, in requestQuery) (result page, err error) {
	var p filter.Predicate
	var sumBy []filter.Field
	p, err = filter.Parse(in.Filter)
	if err != nil {
		return result, err
	}

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
	balances, err := h.Indexer.Balances(ctx, p, in.FilterParams, sumBy, timestampMS)
	if err != nil {
		return result, err
	}

	result.Items = httpjson.Array(balances)
	result.LastPage = true
	result.Next = in
	return result, nil
}

// This type enforces the ordering of JSON fields in API output.
type utxoResp struct {
	Type            string           `json:"type"`
	Purpose         string           `json:"purpose"`
	TransactionID   string           `json:"transaction_id"`
	Position        uint32           `json:"position"`
	AssetID         string           `json:"asset_id"`
	AssetAlias      string           `json:"asset_alias"`
	AssetDefinition *json.RawMessage `json:"asset_definition"`
	AssetTags       *json.RawMessage `json:"asset_tags"`
	AssetIsLocal    string           `json:"asset_is_local"`
	Amount          uint64           `json:"amount"`
	AccountID       string           `json:"account_id"`
	AccountAlias    string           `json:"account_alias"`
	AccountTags     *json.RawMessage `json:"account_tags"`
	ControlProgram  string           `json:"control_program"`
	ReferenceData   *json.RawMessage `json:"reference_data"`
	IsLocal         string           `json:"is_local"`
}

// POST /list-unspent-outputs
func (h *Handler) listUnspentOutputs(ctx context.Context, in requestQuery) (result page, err error) {
	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	// Build the filter predicate.
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

	timestampMS := in.TimestampMS
	if timestampMS == 0 {
		timestampMS = math.MaxInt64
	} else if timestampMS > math.MaxInt64 {
		return result, errors.WithDetail(httpjson.ErrBadRequest, "timestamp is too large")
	}
	outputs, nextAfter, err := h.Indexer.Outputs(ctx, p, in.FilterParams, timestampMS, after, limit)
	if err != nil {
		return result, errors.Wrap(err, "querying outputs")
	}

	resp := make([]*utxoResp, 0, len(outputs))
	for _, o := range outputs {
		var r utxoResp
		err := json.Unmarshal(o, &r)
		if err != nil {
			return result, errors.Wrap(err, "unmarshaling stored utxo")
		}
		resp = append(resp, &r)
	}

	outQuery := in
	outQuery.After = nextAfter.String()
	return page{
		Items:    resp,
		LastPage: len(resp) < limit,
		Next:     outQuery,
	}, nil
}

// listAssets is an http handler for listing assets matching
// an index or an ad-hoc filter.
//
// POST /list-assets
func (h *Handler) listAssets(ctx context.Context, in requestQuery) (page, error) {
	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	// Build the filter predicate.
	p, err := filter.Parse(in.Filter)
	if err != nil {
		return page{}, err
	}
	after := in.After

	// Use the query engine for querying asset tags.
	assets, after, err := h.Indexer.Assets(ctx, p, in.FilterParams, after, limit)
	if err != nil {
		return page{}, errors.Wrap(err, "running asset query")
	}

	result := make([]*assetResponse, 0, len(assets))
	for _, a := range assets {
		var r assetResponse
		err := json.Unmarshal(a, &r)
		if err != nil {
			return page{}, errors.Wrap(err, "unmarshaling stored asset")
		}
		result = append(result, &r)
	}

	out := in
	out.After = after
	return page{
		Items:    httpjson.Array(result),
		LastPage: len(result) < limit,
		Next:     out,
	}, nil
}

// listTxFeeds is an http handler for listing txfeeds. It does not take a filter.
//
// POST /list-transaction-feeds
func (h *Handler) listTxFeeds(ctx context.Context, in requestQuery) (page, error) {
	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	after := in.After

	txfeeds, after, err := h.Indexer.TxFeeds(ctx, after, limit)
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
