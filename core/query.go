package core

import (
	"encoding/json"
	"math"
	"time"

	"golang.org/x/net/context"

	"chain/core/pb"
	"chain/core/query"
	"chain/core/query/filter"
	"chain/errors"
	"chain/net/http/httpjson"
)

func protoParams(params []*pb.FilterParam) []interface{} {
	a := make([]interface{}, len(params))
	for i, p := range params {
		switch p.GetValue().(type) {
		case *pb.FilterParam_String_:
			a[i] = p.GetString_()
		case *pb.FilterParam_Int64:
			a[i] = p.GetInt64()
		case *pb.FilterParam_Bytes:
			a[i] = p.GetBytes()
		case *pb.FilterParam_Bool:
			a[i] = p.GetBool()
		}
	}
	return a
}

// ListAccounts is an http handler for listing accounts matching
// an index or an ad-hoc filter.
func (h *Handler) ListAccounts(ctx context.Context, in *pb.ListAccountsQuery) (*pb.ListAccountsResponse, error) {
	limit := int(in.PageSize)
	if limit == 0 {
		limit = defGenericPageSize
	}

	// Build the filter predicate.
	p, err := filter.Parse(in.Filter)
	if err != nil {
		return nil, errors.Wrap(err, "parsing acc query")
	}
	after := in.After

	// Use the filter engine for querying account tags.
	accounts, after, err := h.Indexer.Accounts(ctx, p, protoParams(in.FilterParams), after, limit)
	if err != nil {
		return nil, errors.Wrap(err, "running acc query")
	}

	result := make([]*pb.Account, 0, len(accounts))
	for _, a := range accounts {
		var resp accountResponse
		err := json.Unmarshal(a, &resp)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshaling indexed account")
		}

		var keys []*pb.Account_Key
		for _, k := range resp.Keys {
			adp := make([][]byte, 0, len(k.AccountDerivationPath))
			for _, p := range k.AccountDerivationPath {
				adp = append(adp, p)
			}
			keys = append(keys, &pb.Account_Key{
				RootXpub:              k.RootXPub[:],
				AccountXpub:           k.AccountXPub[:],
				AccountDerivationPath: adp,
			})
		}

		result = append(result, &pb.Account{
			Id:     resp.ID,
			Alias:  resp.Alias,
			Keys:   keys,
			Quorum: int32(resp.Quorum),
			Tags:   resp.Tags,
		})
	}

	// Pull in the accounts by the IDs
	out := in
	out.After = after
	return &pb.ListAccountsResponse{
		Items:    result,
		LastPage: len(result) < limit,
		Next:     out,
	}, nil
}

// ListAssets is an http handler for listing assets matching
// an index or an ad-hoc filter.
func (h *Handler) ListAssets(ctx context.Context, in *pb.ListAssetsQuery) (*pb.ListAssetsResponse, error) {
	limit := int(in.PageSize)
	if limit == 0 {
		limit = defGenericPageSize
	}

	// Build the filter predicate.
	p, err := filter.Parse(in.Filter)
	if err != nil {
		return nil, err
	}
	after := in.After

	// Use the query engine for querying asset tags.
	assets, after, err := h.Indexer.Assets(ctx, p, protoParams(in.FilterParams), after, limit)
	if err != nil {
		return nil, errors.Wrap(err, "running asset query")
	}

	result := make([]*pb.Asset, 0, len(assets))
	for _, a := range assets {
		var resp assetResponse
		err := json.Unmarshal(a, &resp)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshaling indexed asset")
		}

		var keys []*pb.Asset_Key
		for _, k := range resp.Keys {
			adp := make([][]byte, 0, len(k.AssetDerivationPath))
			for _, p := range k.AssetDerivationPath {
				adp = append(adp, p)
			}
			keys = append(keys, &pb.Asset_Key{
				RootXpub:            k.RootXPub[:],
				AssetPubkey:         k.AssetPubkey,
				AssetDerivationPath: adp,
			})
		}

		result = append(result, &pb.Asset{
			Id:              resp.ID[:],
			Alias:           resp.Alias,
			IssuanceProgram: resp.IssuanceProgram,
			Keys:            keys,
			Quorum:          int32(resp.Quorum),
			Definition:      resp.Definition,
			Tags:            resp.Tags,
			IsLocal:         bool(resp.IsLocal),
		})
	}

	out := in
	out.After = after
	return &pb.ListAssetsResponse{
		Items:    result,
		LastPage: len(result) < limit,
		Next:     out,
	}, nil
}

func (h *Handler) ListBalances(ctx context.Context, in *pb.ListBalancesQuery) (*pb.ListBalancesResponse, error) {
	var sumBy []filter.Field
	p, err := filter.Parse(in.Filter)
	if err != nil {
		return nil, err
	}

	// Since an empty SumBy yields a meaningless result, we'll provide a
	// sensible default here.
	if len(in.SumBy) == 0 {
		in.SumBy = []string{"asset_alias", "asset_id"}
	}

	for _, field := range in.SumBy {
		f, err := filter.ParseField(field)
		if err != nil {
			return nil, err
		}
		sumBy = append(sumBy, f)
	}

	timestampMS := in.Timestamp
	if timestampMS == 0 {
		timestampMS = math.MaxInt64
	} else if timestampMS > math.MaxInt64 {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "timestamp is too large")
	}

	// TODO(jackson): paginate this endpoint.
	balances, err := h.Indexer.Balances(ctx, p, protoParams(in.FilterParams), sumBy, timestampMS)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(httpjson.Array(balances))
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &pb.ListBalancesResponse{
		Items:    data,
		LastPage: true,
		Next:     in,
	}, nil
}

// ListTxs is an http handler for listing transactions matching
// an index or an ad-hoc filter.
func (h *Handler) ListTxs(ctx context.Context, in *pb.ListTxsQuery) (*pb.ListTxsResponse, error) {
	var (
		timeout time.Duration
		err     error
	)

	if in.Timeout != "" {
		timeout, err = time.ParseDuration(in.Timeout)
	}
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if timeout != 0 {
		var c context.CancelFunc
		ctx, c = context.WithTimeout(ctx, timeout)
		defer c()
	}

	limit := int(in.PageSize)
	if limit == 0 {
		limit = defGenericPageSize
	}

	// Build the filter predicate.
	p, err := filter.Parse(in.Filter)
	if err != nil {
		return nil, err
	}

	endTimeMS := in.EndTime
	if endTimeMS == 0 {
		endTimeMS = math.MaxInt64
	} else if endTimeMS > math.MaxInt64 {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "end timestamp is too large")
	}

	var after query.TxAfter
	// Either parse the provided `after` or look one up for the time range.
	if in.After != "" {
		after, err = query.DecodeTxAfter(in.After)
		if err != nil {
			return nil, errors.Wrap(err, "decoding `after`")
		}
	} else {
		after, err = h.Indexer.LookupTxAfter(ctx, in.StartTime, endTimeMS)
		if err != nil {
			return nil, err
		}
	}

	txns, nextAfter, err := h.Indexer.Transactions(ctx, p, protoParams(in.FilterParams), after, limit, in.AscendingWithLongPoll)
	if err != nil {
		return nil, errors.Wrap(err, "running tx query")
	}

	data, err := json.Marshal(httpjson.Array(txns))
	if err != nil {
		return nil, errors.Wrap(err)
	}

	out := in
	out.After = nextAfter.String()
	return &pb.ListTxsResponse{
		Items:    data,
		LastPage: len(txns) < limit,
		Next:     out,
	}, nil
}

// ListTxFeeds is an http handler for listing txfeeds. It does not take a filter.
func (h *Handler) ListTxFeeds(ctx context.Context, in *pb.ListTxFeedsQuery) (*pb.ListTxFeedsResponse, error) {
	limit := int(in.PageSize)
	if limit == 0 {
		limit = defGenericPageSize
	}
	after := in.After

	txfeeds, after, err := h.Indexer.TxFeeds(ctx, after, limit)
	if err != nil {
		return nil, errors.Wrap(err, "running txfeed query")
	}

	var pbFeeds []*pb.TxFeed
	for _, f := range txfeeds {
		proto := &pb.TxFeed{
			Id:     f.ID,
			Filter: f.Filter,
			After:  f.After,
		}
		if f.Alias != nil {
			proto.Alias = *f.Alias
		}
		pbFeeds = append(pbFeeds, proto)
	}

	out := in
	out.After = after
	return &pb.ListTxFeedsResponse{
		Items:    pbFeeds,
		LastPage: len(txfeeds) < limit,
		Next:     out,
	}, nil
}

func (h *Handler) ListUnspentOutputs(ctx context.Context, in *pb.ListUnspentOutputsQuery) (*pb.ListUnspentOutputsResponse, error) {
	p, err := filter.Parse(in.Filter)
	if err != nil {
		return nil, err
	}

	limit := int(in.PageSize)
	if limit == 0 {
		limit = defGenericPageSize
	}

	var after *query.OutputsAfter
	if in.After != "" {
		after, err = query.DecodeOutputsAfter(in.After)
		if err != nil {
			return nil, errors.Wrap(err, "decoding `after`")
		}
	}

	timestampMS := in.Timestamp
	if timestampMS == 0 {
		timestampMS = math.MaxInt64
	} else if timestampMS > math.MaxInt64 {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "timestamp is too large")
	}
	outputs, nextAfter, err := h.Indexer.Outputs(ctx, p, protoParams(in.FilterParams), timestampMS, after, limit)
	if err != nil {
		return nil, errors.Wrap(err, "querying outputs")
	}

	data, err := json.Marshal(httpjson.Array(outputs))
	if err != nil {
		return nil, errors.Wrap(err)
	}

	outQuery := in
	outQuery.After = nextAfter.String()
	return &pb.ListUnspentOutputsResponse{
		Items:    data,
		LastPage: len(outputs) < limit,
		Next:     outQuery,
	}, nil
}
