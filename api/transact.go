package api

import (
	"bytes"
	"sync"
	"time"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/encoding/json"
	"chain/errors"
	"chain/fedchain-sandbox/wire"
	"chain/metrics"
)

type buildReq struct {
	PrevTx  *asset.Tx `json:"previous_transaction"`
	Inputs  []utxodb.Input
	Outputs []asset.Output
	ResTime time.Duration `json:"reservation_duration"`
}

type transferReq struct {
	Inputs  []utxodb.Input
	Outputs []asset.Output
}

// POST /v3/assets/:assetID/issue
func issueAsset(ctx context.Context, assetID string, outs []asset.Output) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	template, err := asset.Issue(ctx, assetID, outs)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{"template": template}
	return ret, nil
}

// POST /v3/assets/transfer
func transferAssets(ctx context.Context, x transferReq) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	template, err := asset.Transfer(ctx, x.Inputs, x.Outputs)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{"template": template}
	return ret, nil
}

// POST /v3/transact/build
func build(ctx context.Context, buildReqs []buildReq) interface{} {
	defer metrics.RecordElapsed(time.Now())

	responses := make([]interface{}, len(buildReqs))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			defer wg.Done()

			dbtx, ctx, err := pg.Begin(ctx)
			if err != nil {
				responses[i], _ = errInfo(err)
				return
			}
			defer dbtx.Rollback()

			tpl, err := asset.Build(ctx, buildReqs[i].PrevTx, buildReqs[i].Inputs, buildReqs[i].Outputs, buildReqs[i].ResTime)
			if err != nil {
				responses[i], _ = errInfo(err)
				return
			}

			err = dbtx.Commit()
			if err != nil {
				responses[i], _ = errInfo(err)
				return
			}

			responses[i] = map[string]interface{}{"template": tpl}
		}(i)
	}

	wg.Wait()
	return responses
}

// POST /v3/assets/trade
func tradeAssets(ctx context.Context, x struct {
	PreviousTx *asset.Tx `json:"previous_transaction"`
	Inputs     []utxodb.Input
	Outputs    []asset.Output
}) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	template, err := asset.Trade(ctx, x.PreviousTx, x.Inputs, x.Outputs)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{"template": template}
	return ret, nil
}

// POST /v3/manager-nodes/transact/finalize
func walletFinalize(ctx context.Context, tpl *asset.Tx) (interface{}, error) {
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

	var buf bytes.Buffer
	tx.Serialize(&buf)

	ret := map[string]interface{}{
		"transaction_id":  tx.TxSha().String(),
		"raw_transaction": json.HexBytes(buf.Bytes()),
	}
	return ret, nil
}

// POST /v3/assets/cancel-reservation
func cancelReservation(ctx context.Context, x struct {
	Transaction json.HexBytes
}) error {
	tx := wire.NewMsgTx()
	err := tx.Deserialize(bytes.NewReader(x.Transaction))
	if err != nil {
		return errors.Wrap(asset.ErrBadTxHex)
	}

	asset.CancelReservations(ctx, tx.OutPoints())
	return nil
}

// POST /v3/assets/transfer-batch
func batchTransfer(ctx context.Context, x struct{ Transfers []transferReq }) interface{} {
	defer metrics.RecordElapsed(time.Now())

	responses := make([]interface{}, len(x.Transfers))
	var wg sync.WaitGroup
	wg.Add(len(responses))
	for i := 0; i < len(responses); i++ {
		go func(i int) {
			resp, err := transferAssets(ctx, x.Transfers[i])
			if err != nil {
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

// POST /v3/wallets/transact/finalize-batch
func batchFinalize(ctx context.Context, x struct{ Transactions []*asset.Tx }) interface{} {
	defer metrics.RecordElapsed(time.Now())

	responses := make([]interface{}, len(x.Transactions))
	var wg sync.WaitGroup
	wg.Add(len(responses))
	for i := range responses {
		go func(i int) {
			resp, err := walletFinalize(ctx, x.Transactions[i])
			if err != nil {
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
