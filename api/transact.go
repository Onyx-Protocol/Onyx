package api

import (
	"sync"
	"time"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/issuer"
	"chain/api/txbuilder"
	"chain/database/pg"
	"chain/fedchain/bc"
	"chain/metrics"
	"chain/net/http/reqid"
	"chain/net/trace/span"
)

// POST /v3/assets/:assetID/issue
func issueAsset(ctx context.Context, assetIDStr string, reqDests []*Destination) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)
	if err := assetAuthz(ctx, assetIDStr); err != nil {
		return nil, err
	}

	var assetID bc.AssetID
	err := assetID.UnmarshalText([]byte(assetIDStr))
	if err != nil {
		return nil, err
	}

	// Where asset_ids are not specified in destinations - and even
	// where they are - use the one passed in above.
	for _, dest := range reqDests {
		dest.AssetID = &assetID
	}

	dests := make([]*txbuilder.Destination, 0, len(reqDests))
	for _, reqDest := range reqDests {
		parsed, err := reqDest.parse(ctx)
		if err != nil {
			return nil, err
		}
		dests = append(dests, parsed)
	}

	template, err := issuer.Issue(ctx, assetIDStr, dests)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{"template": template}
	return ret, nil
}

func buildSingle(ctx context.Context, req *BuildRequest) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback(ctx)

	prevTx, sources, destinations, err := req.parse(ctx)
	if err != nil {
		return nil, err
	}

	tpl, err := txbuilder.Build(ctx, prevTx, sources, destinations, req.Metadata, req.ResTime)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"template": tpl}, nil
}

// POST /v3/transact/build
// Idempotent
func build(ctx context.Context, buildReqs []*BuildRequest) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	if err := buildAuthz(ctx, buildReqs...); err != nil {
		return nil, err
	}

	responses := make([]interface{}, len(buildReqs))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			defer wg.Done()
			resp, err := buildSingle(reqid.NewSubContext(ctx, reqid.New()), buildReqs[i])
			if err != nil {
				logHTTPError(ctx, err)
				responses[i], _ = errInfo(err)
			} else {
				responses[i] = resp
			}
		}(i)
	}

	wg.Wait()
	return responses, nil
}

// POST /v3/manager-nodes/transact/finalize
// Idempotent
func submitSingle(ctx context.Context, tpl *Template) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	parsed, err := tpl.parse(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := asset.FinalizeTx(ctx, parsed)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"transaction_id":  tx.Hash.String(),
		"raw_transaction": tx,
	}
	return ret, nil
}

// TODO(bobg): allow caller to specify reservation by (encrypted) id?
// POST /v3/assets/cancel-reservation
// Idempotent
func cancelReservation(ctx context.Context, x struct{ Transaction bc.Tx }) error {
	var outpoints []bc.Outpoint
	for _, input := range x.Transaction.Inputs {
		outpoints = append(outpoints, input.Previous)
	}
	return asset.CancelReservations(ctx, outpoints)
}

// POST /v3/transact/submit
// Idempotent
func submit(ctx context.Context, x struct{ Transactions []*Template }) interface{} {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	responses := make([]interface{}, len(x.Transactions))
	var wg sync.WaitGroup
	wg.Add(len(responses))
	for i := range responses {
		go func(i int) {
			resp, err := submitSingle(reqid.NewSubContext(ctx, reqid.New()), x.Transactions[i])
			if err != nil {
				logHTTPError(ctx, err)
				responses[i], _ = errInfo(err)
			} else {
				responses[i] = resp
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	return responses
}
