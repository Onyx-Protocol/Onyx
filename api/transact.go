package api

import (
	"sync"
	"time"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/fedchain/bc"
	"chain/metrics"
	"chain/net/http/reqid"
)

type buildReq struct {
	PrevTx  *asset.Tx `json:"previous_transaction"`
	Inputs  []utxodb.Input
	Outputs []*asset.Output
	ResTime time.Duration `json:"reservation_duration"`
}

// POST /v3/assets/:assetID/issue
func issueAsset(ctx context.Context, assetID string, outs []*asset.Output) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	if err := assetAuthz(ctx, assetID); err != nil {
		return nil, err
	}
	template, err := asset.Issue(ctx, assetID, outs)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{"template": template}
	return ret, nil
}

func buildSingle(ctx context.Context, req buildReq) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	tpl, err := asset.Build(ctx, req.PrevTx, req.Inputs, req.Outputs, req.ResTime)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"template": tpl}, nil
}

// POST /v3/transact/build
func build(ctx context.Context, buildReqs []buildReq) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
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
func submitSingle(ctx context.Context, tpl *asset.Tx) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	// TODO(kr): validate

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	tx, err := asset.FinalizeTx(ctx, tpl)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"transaction_id":  tx.Hash().String(),
		"raw_transaction": tx,
	}
	return ret, nil
}

// POST /v3/assets/cancel-reservation
func cancelReservation(ctx context.Context, x struct{ Transaction bc.Tx }) error {
	var outpoints []bc.Outpoint
	for _, input := range x.Transaction.Inputs {
		outpoints = append(outpoints, input.Previous)
	}
	asset.CancelReservations(ctx, outpoints)
	return nil
}

// POST /v3/transact/submit
func submit(ctx context.Context, x struct{ Transactions []*asset.Tx }) interface{} {
	defer metrics.RecordElapsed(time.Now())

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
