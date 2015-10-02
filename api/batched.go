package api

import (
	"sync"
	"time"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/metrics"
)

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
