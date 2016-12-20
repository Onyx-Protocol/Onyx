package core

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"chain-stealth/core/query"
	"chain-stealth/core/query/filter"
	"chain-stealth/errors"
	"chain-stealth/net/http/httpjson"
)

// These types enforce the ordering of JSON fields in API output.
type (
	txinResp struct {
		Type              interface{} `json:"type"`
		AssetID           interface{} `json:"asset_id,omitempty"`
		AssetIDCommitment interface{} `json:"asset_id_commitment,omitempty"`
		AssetAlias        interface{} `json:"asset_alias,omitempty"`
		AssetDefinition   interface{} `json:"asset_definition"`
		AssetTags         interface{} `json:"asset_tags,omitempty"`
		AssetIsLocal      interface{} `json:"asset_is_local"`
		Amount            interface{} `json:"amount,omitempty"`
		AmountCommitment  interface{} `json:"amount_commitment,omitempty"`
		IssuanceProgram   interface{} `json:"issuance_program,omitempty"`
		SpentOutput       interface{} `json:"spent_output,omitempty"`
		*txAccount
		ReferenceData interface{} `json:"reference_data"`
		IsLocal       interface{} `json:"is_local"`
		Confidential  interface{} `json:"confidential"`
		Readable      interface{} `json:"readable"`
	}
	txoutResp struct {
		Type              interface{} `json:"type"`
		Purpose           interface{} `json:"purpose,omitempty"`
		Position          interface{} `json:"position"`
		AssetID           interface{} `json:"asset_id,omitempty"`
		AssetIDCommitment interface{} `json:"asset_id_commitment,omitempty"`
		AssetAlias        interface{} `json:"asset_alias,omitempty"`
		AssetDefinition   interface{} `json:"asset_definition,omitempty"`
		AssetTags         interface{} `json:"asset_tags,omitempty"`
		AssetIsLocal      interface{} `json:"asset_is_local"`
		Amount            interface{} `json:"amount,omitempty"`
		AmountCommitment  interface{} `json:"amount_commitment,omitempty"`
		*txAccount
		ControlProgram interface{} `json:"control_program"`
		ReferenceData  interface{} `json:"reference_data"`
		IsLocal        interface{} `json:"is_local"`
		Confidential   interface{} `json:"confidential"`
		Readable       interface{} `json:"readable"`
	}
	txResp struct {
		ID            interface{} `json:"id"`
		Timestamp     interface{} `json:"timestamp"`
		BlockID       interface{} `json:"block_id"`
		BlockHeight   interface{} `json:"block_height"`
		Position      interface{} `json:"position"`
		ReferenceData interface{} `json:"reference_data"`
		IsLocal       interface{} `json:"is_local"`
		Inputs        interface{} `json:"inputs"`
		Outputs       interface{} `json:"outputs"`
	}
	txAccount struct {
		AccountID    interface{} `json:"account_id"`
		AccountAlias interface{} `json:"account_alias,omitempty"`
		AccountTags  interface{} `json:"account_tags"`
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

	limit := defGenericPageSize
	txns, nextAfter, err := h.Indexer.Transactions(ctx, p, in.FilterParams, after, limit, in.AscLongPoll)
	if err != nil {
		return result, errors.Wrap(err, "running tx query")
	}

	resp := make([]*txResp, 0, len(txns))
	for _, t := range txns {
		tjson, ok := t.(*json.RawMessage)
		if !ok {
			return result, fmt.Errorf("unexpected type %T in Indexer.Transactions output", t)
		}
		if tjson == nil {
			return result, fmt.Errorf("unexpected nil in Indexer.Transactions output")
		}
		var tx map[string]interface{}
		err = json.Unmarshal(*tjson, &tx)
		if err != nil {
			return result, errors.Wrap(err, "decoding Indexer.Transactions output")
		}

		inp, ok := tx["inputs"].([]interface{})
		if !ok {
			return result, fmt.Errorf("unexpected type %T for inputs in Indexer.Transactions output", tx["inputs"])
		}

		var inputs []map[string]interface{}
		for i, in := range inp {
			input, ok := in.(map[string]interface{})
			if !ok {
				return result, fmt.Errorf("unexpected type %T for input %d in Indexer.Transactions output", in, i)
			}
			inputs = append(inputs, input)
		}

		outp, ok := tx["outputs"].([]interface{})
		if !ok {
			return result, fmt.Errorf("unexpected type %T for outputs in Indexer.Transactions output", tx["outputs"])
		}

		var outputs []map[string]interface{}
		for i, out := range outp {
			output, ok := out.(map[string]interface{})
			if !ok {
				return result, fmt.Errorf("unexpected type %T for output %d in Indexer.Transactions output", out, i)
			}
			outputs = append(outputs, output)
		}

		inResps := make([]*txinResp, 0, len(inputs))
		for _, in := range inputs {
			r := &txinResp{
				Type:              in["type"],
				AssetID:           in["asset_id"],
				AssetIDCommitment: in["asset_id_commitment"],
				AssetAlias:        in["asset_alias"],
				AssetDefinition:   in["asset_definition"],
				AssetTags:         in["asset_tags"],
				AssetIsLocal:      in["asset_is_local"],
				Amount:            in["amount"],
				AmountCommitment:  in["amount_commitment"],
				IssuanceProgram:   in["issuance_program"],
				SpentOutput:       in["spent_output"],
				txAccount:         txAccountFromMap(in),
				ReferenceData:     in["reference_data"],
				IsLocal:           in["is_local"],
				Confidential:      in["confidential"],
				Readable:          in["readable"],
			}
			inResps = append(inResps, r)
		}
		outResps := make([]*txoutResp, 0, len(outputs))
		for _, out := range outputs {
			r := &txoutResp{
				Type:              out["type"],
				Purpose:           out["purpose"],
				Position:          out["position"],
				AssetID:           out["asset_id"],
				AssetAlias:        out["asset_alias"],
				AssetDefinition:   out["asset_definition"],
				AssetTags:         out["asset_tags"],
				AssetIsLocal:      out["asset_is_local"],
				AssetIDCommitment: out["asset_id_commitment"],
				Amount:            out["amount"],
				AmountCommitment:  out["amount_commitment"],
				txAccount:         txAccountFromMap(out),
				ControlProgram:    out["control_program"],
				ReferenceData:     out["reference_data"],
				IsLocal:           out["is_local"],
				Confidential:      out["confidential"],
				Readable:          out["readable"],
			}
			outResps = append(outResps, r)
		}
		r := &txResp{
			ID:            tx["id"],
			Timestamp:     tx["timestamp"],
			BlockID:       tx["block_id"],
			BlockHeight:   tx["block_height"],
			Position:      tx["position"],
			ReferenceData: tx["reference_data"],
			IsLocal:       tx["is_local"],
			Inputs:        inResps,
			Outputs:       outResps,
		}
		resp = append(resp, r)
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
	limit := defGenericPageSize

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
		var orderedKeys []accountKey
		keys, ok := a["keys"].([]interface{})
		if ok {
			for _, key := range keys {
				mapKey, ok := key.(map[string]interface{})
				if !ok {
					continue
				}
				orderedKeys = append(orderedKeys, accountKey{
					RootXPub:              mapKey["root_xpub"],
					AccountXPub:           mapKey["account_xpub"],
					AccountDerivationPath: mapKey["account_derivation_path"],
				})
			}
		}
		r := &accountResponse{
			ID:     a["id"],
			Alias:  a["alias"],
			Keys:   orderedKeys,
			Quorum: a["quorum"],
			Tags:   a["tags"],
		}
		result = append(result, r)
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
	Type            interface{} `json:"type"`
	Purpose         interface{} `json:"purpose"`
	TransactionID   interface{} `json:"transaction_id"`
	Position        interface{} `json:"position"`
	AssetID         interface{} `json:"asset_id,omitempty"`
	AssetAlias      interface{} `json:"asset_alias,omitempty"`
	AssetDefinition interface{} `json:"asset_definition,omitempty"`
	AssetTags       interface{} `json:"asset_tags,omitempty"`
	AssetIsLocal    interface{} `json:"asset_is_local"`
	AssetCommitment interface{} `json:"asset_commitment,omitempty"`
	Amount          interface{} `json:"amount,omitempty"`
	ValueCommitment interface{} `json:"value_commitment,omitempty"`
	AccountID       interface{} `json:"account_id"`
	AccountAlias    interface{} `json:"account_alias"`
	AccountTags     interface{} `json:"account_tags"`
	ControlProgram  interface{} `json:"control_program"`
	ReferenceData   interface{} `json:"reference_data"`
	IsLocal         interface{} `json:"is_local"`
	Confidential    interface{} `json:"confidential"`
	Readable        interface{} `json:"readable"`
}

// POST /list-unspent-outputs
func (h *Handler) listUnspentOutputs(ctx context.Context, in requestQuery) (result page, err error) {
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
	limit := defGenericPageSize
	outputs, nextAfter, err := h.Indexer.Outputs(ctx, p, in.FilterParams, timestampMS, after, limit)
	if err != nil {
		return result, errors.Wrap(err, "querying outputs")
	}

	resp := make([]*utxoResp, 0, len(outputs))
	for _, o := range outputs {
		ojson, ok := o.(*json.RawMessage)
		if !ok {
			return result, fmt.Errorf("unexpected type %T in Indexer.Outputs output", o)
		}
		if ojson == nil {
			return result, fmt.Errorf("unexpected nil in Indexer.Outputs output")
		}
		var out map[string]interface{}
		err = json.Unmarshal(*ojson, &out)
		if err != nil {
			return result, errors.Wrap(err, "decoding Indexer.Outputs output")
		}
		r := &utxoResp{
			Type:            out["type"],
			Purpose:         out["purpose"],
			TransactionID:   out["transaction_id"],
			Position:        out["position"],
			AssetID:         out["asset_id"],
			AssetAlias:      out["asset_alias"],
			AssetDefinition: out["asset_definition"],
			AssetTags:       out["asset_tags"],
			AssetIsLocal:    out["asset_is_local"],
			AssetCommitment: out["asset_commitment"],
			Amount:          out["amount"],
			ValueCommitment: out["value_commitment"],
			AccountID:       out["account_id"],
			AccountAlias:    out["account_alias"],
			AccountTags:     out["account_tags"],
			ControlProgram:  out["control_program"],
			ReferenceData:   out["reference_data"],
			IsLocal:         out["is_local"],
			Confidential:    out["confidential"],
			Readable:        out["readable"],
		}
		resp = append(resp, r)
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
	limit := defGenericPageSize

	// Build the filter predicate.
	p, err := filter.Parse(in.Filter)
	if err != nil {
		return page{}, err
	}
	after := in.After

	// Use the query engine for querying asset tags.
	var assets []map[string]interface{}
	assets, after, err = h.Indexer.Assets(ctx, p, in.FilterParams, after, limit)
	if err != nil {
		return page{}, errors.Wrap(err, "running asset query")
	}

	result := make([]*assetResponse, 0, len(assets))
	for _, a := range assets {
		var orderedKeys []assetKey
		keys, ok := a["keys"].([]interface{})
		if ok {
			for _, key := range keys {
				mapKey, ok := key.(map[string]interface{})
				if !ok {
					continue
				}
				orderedKeys = append(orderedKeys, assetKey{
					AssetPubkey:         mapKey["asset_pubkey"],
					RootXPub:            mapKey["root_xpub"],
					AssetDerivationPath: mapKey["asset_derivation_path"],
				})
			}
		}
		r := &assetResponse{
			ID:              a["id"],
			IssuanceProgram: a["issuance_program"],
			Keys:            orderedKeys,
			Quorum:          a["quorum"],
			Definition:      a["definition"],
			Tags:            a["tags"],
			IsLocal:         a["is_local"],
		}
		if alias, ok := a["alias"].(string); ok && alias != "" {
			r.Alias = &alias
		}
		result = append(result, r)
	}

	out := in
	out.After = after
	return page{
		Items:    httpjson.Array(result),
		LastPage: len(result) < limit,
		Next:     out,
	}, nil
}

func txAccountFromMap(m map[string]interface{}) *txAccount {
	if _, ok := m["account_id"]; !ok {
		return nil
	}
	return &txAccount{
		AccountID:    m["account_id"],
		AccountAlias: m["account_alias"],
		AccountTags:  m["account_tags"],
	}
}

// listTxFeeds is an http handler for listing txfeeds. It does not take a filter.
//
// POST /list-transaction-feeds
func (h *Handler) listTxFeeds(ctx context.Context, in requestQuery) (page, error) {
	limit := defGenericPageSize
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
